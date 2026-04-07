package repository

import (
	"context"
	"errors"
	"products/internal/gen"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type PromoCodesRepository struct {
	conn *sqlx.DB
}

func NewPromoCodesRepository(db *sqlx.DB) *PromoCodesRepository {
	return &PromoCodesRepository{conn: db}
}

var ErrSamePromoExists = errors.New("Promo with same code exists")

type PromoCodeToCreateRepository struct {
	Active         *bool
	Code           string
	DiscountType   gen.DiscountType `db:"discount_type"`
	DiscountValue  decimal.Decimal  `db:"discount_value"`
	MaxUses        int              `db:"max_uses"`
	MinOrderAmount decimal.Decimal  `db:"min_order_amount"`
	ValidFrom      time.Time        `db:"valid_from"`
	ValidUntil     time.Time        `db:"valid_until"`
}

type PromoCodeRepository struct {
	PromoCodeToCreateRepository
	Id          uuid.UUID
	CurrentUses int `db:"current_uses"`
	Active      bool
}

func (db *PromoCodesRepository) CreatePromoCode(
	ctx context.Context, values PromoCodeToCreateRepository) (*PromoCodeRepository, error) {
	query, err := db.conn.PrepareNamed(`
		INSERT INTO promo_codes (
			active, code, discount_type, discount_value, max_uses, min_order_amount, valid_from, valid_until
		)
		VALUES (
			:active, :code, :discount_type, :discount_value, :max_uses, :min_order_amount, :valid_from, :valid_until
		)
		RETURNING *
	`)
	if err != nil {
		return nil, err
	}
	defer query.Close()
	var product PromoCodeRepository
	err = query.GetContext(ctx, &product, values)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, ErrSamePromoExists
		}
	}
	return &product, err
}
