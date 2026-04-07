package service

import (
	"context"
	"errors"
	"products/internal/gen"
	"products/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PromoCodesService struct {
	db *repository.PromoCodesRepository
}

func NewPromoCodesService(db *repository.PromoCodesRepository) *PromoCodesService {
	return &PromoCodesService{db: db}
}

type PromoCodeToCreateService struct {
	Active         *bool
	Code           string
	DiscountType   gen.DiscountType
	DiscountValue  decimal.Decimal
	MaxUses        int
	MinOrderAmount decimal.Decimal
	ValidFrom      time.Time
	ValidUntil     time.Time
}

type PromoCodeService struct {
	Id             uuid.UUID
	CurrentUses    int
	Active         bool
	Code           string
	DiscountType   gen.DiscountType
	DiscountValue  decimal.Decimal
	MaxUses        int
	MinOrderAmount decimal.Decimal
	ValidFrom      time.Time
	ValidUntil     time.Time
}

func (srvc *PromoCodesService) CreatePromoCode(
	ctx context.Context, values PromoCodeToCreateService) (*PromoCodeService, error) {
	promo, err := srvc.db.CreatePromoCode(ctx, repository.PromoCodeToCreateRepository(values))
	if err != nil {
		if errors.Is(err, repository.ErrSamePromoExists) {
			return nil, ErrSamePromoExists
		}
		return nil, err
	}
	return &PromoCodeService{
		Id:             promo.Id,
		CurrentUses:    promo.CurrentUses,
		Active:         promo.Active,
		Code:           promo.Code,
		DiscountType:   promo.DiscountType,
		DiscountValue:  promo.DiscountValue,
		MaxUses:        promo.MaxUses,
		MinOrderAmount: promo.MinOrderAmount,
		ValidFrom:      promo.ValidFrom,
		ValidUntil:     promo.ValidUntil,
	}, nil
}
