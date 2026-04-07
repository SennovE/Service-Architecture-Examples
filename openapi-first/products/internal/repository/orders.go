package repository

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"products/internal/gen"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type OrdersRepository struct {
	conn *sqlx.DB
}

type OrdersTransaction struct {
	conn *sqlx.Tx
}

func NewOrdersRepository(db *sqlx.DB) *OrdersRepository {
	return &OrdersRepository{conn: db}
}

func (db *OrdersRepository) BeginTransaction() (*OrdersTransaction, error) {
	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, err
	}
	return &OrdersTransaction{conn: tx}, nil
}

func (tx *OrdersTransaction) Rollback() {
	tx.conn.Rollback()
}

func (tx *OrdersTransaction) Commit() {
	tx.conn.Commit()
}

var (
	ErrOrderNotFound         = errors.New("order not found")
	ErrOrderCreationNotFound = errors.New("user operation not found")
	ErrPromoCodeNotFound     = errors.New("promo code not found")
)

type OrderRepository struct {
	Id             uuid.UUID
	CreatedAt      *time.Time      `db:"created_at"`
	UpdatedAt      *time.Time      `db:"updated_at,omitempty"`
	DiscountAmount decimal.Decimal `db:"discount_amount"`
	PromoCodeId    *uuid.UUID      `db:"promo_code_id"`
	Status         gen.OrderStatus
	TotalAmount    decimal.Decimal `db:"total_amount"`
	UserID         uuid.UUID       `db:"user_id"`
}

type ProductStockToBuyRepository struct {
	Id    uuid.UUID
	Stock int
}

type ProductToBuyRepository struct {
	ProductStockToBuyRepository
	Price  decimal.Decimal
	Status gen.ProductStatus
}

type OrderItemToCreateRepository struct {
	PriceAtOrder decimal.Decimal `db:"price_at_order"`
	ProductId    uuid.UUID       `db:"product_id"`
	Quantity     int
}

type OrderItemRepository struct {
	Id uuid.UUID
	OrderItemToCreateRepository
}

func (db *OrdersRepository) GetOrderById(ctx context.Context, id uuid.UUID) (*OrderRepository, error) {
	query := "SELECT * FROM orders WHERE id = $1"
	var order OrderRepository
	err := db.conn.GetContext(ctx, &order, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
	}
	return &order, err
}

func (db *OrdersRepository) GetOrderItemsByOrderId(ctx context.Context, orderId uuid.UUID) (*[]OrderItemRepository, error) {
	query := `
		SELECT id, price_at_order, product_id, quantity
		FROM order_items
		WHERE order_id = $1
	`
	var orders []OrderItemRepository
	err := db.conn.SelectContext(ctx, &orders, query, orderId)
	return &orders, err
}

func (tx *OrdersTransaction) GetLastOperationTime(ctx context.Context, userID uuid.UUID, operationType string) (*time.Time, error) {
	query := `
		SELECT created_at FROM user_operations
		WHERE user_id = $1 AND operation_type = $2
		ORDER BY created_at DESC
		LIMIT 1
	`
	var createdTime time.Time
	err := tx.conn.GetContext(ctx, &createdTime, query, userID, operationType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
	}
	return &createdTime, err
}

func (tx *OrdersTransaction) GetUserActiveOrder(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	query := `
		SELECT id FROM orders
		WHERE user_id = $1 AND (status = 'CREATED' OR status = 'PAYMENT_PENDING')
		LIMIT 1
	`
	var id uuid.UUID
	err := tx.conn.GetContext(ctx, &id, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

func (tx *OrdersTransaction) GetForUpdateProductsByIds(
	ctx context.Context, ids []uuid.UUID) (*[]ProductToBuyRepository, error) {
	query := `
		SELECT id, stock, status, price FROM products
		WHERE id IN (?) AND status != 'ARCHIVED'
		FOR UPDATE
	`
	query, args, err := sqlx.In(query, ids)
	if err != nil {
		return nil, err
	}
	query = tx.conn.Rebind(query)
	var orders []ProductToBuyRepository
	err = tx.conn.SelectContext(ctx, &orders, query, args...)
	return &orders, err
}

func (tx *OrdersTransaction) UpdateStockByIds(ctx context.Context, products []ProductStockToBuyRepository) error {
	if len(products) == 0 {
		return nil
	}

	args := make([]any, 0, len(products)*2)
	whenClauses := bytes.NewBufferString("")
	inPlaceholders := make([]string, 0, len(products))

	for i, item := range products {
		idArg := i*2 + 1
		stockArg := i*2 + 2

		fmt.Fprintf(whenClauses, " WHEN $%d THEN $%d", idArg, stockArg)

		args = append(args, item.Id, item.Stock)
		inPlaceholders = append(inPlaceholders, fmt.Sprintf("$%d", idArg))
	}

	query := fmt.Sprintf(`
  		UPDATE products
  		SET stock = stock - CASE id
   			%s
   			ELSE stock
  		END
  		WHERE id IN (%s)
 	`, whenClauses.String(), strings.Join(inPlaceholders, ", "))

	_, err := tx.conn.ExecContext(ctx, query, args...)
	return err
}

func (tx *OrdersTransaction) GetPromoCodeByCode(ctx context.Context, promoCode string) (*PromoCodeRepository, error) {
	query := "SELECT * FROM promo_codes WHERE code = $1 FOR UPDATE"
	var promo PromoCodeRepository
	err := tx.conn.GetContext(ctx, &promo, query, promoCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromoCodeNotFound
		}
	}
	return &promo, err
}

func (tx *OrdersTransaction) IncrementPromoUsers(ctx context.Context, promoCode string) error {
	query := `
		UPDATE promo_codes
		SET current_uses = current_uses + 1
		WHERE code = $1
	`
	_, err := tx.conn.ExecContext(ctx, query, promoCode)
	return err
}

func (tx *OrdersTransaction) CreateOrder(
	ctx context.Context, userID uuid.UUID, promoId *uuid.UUID, totalAmount, discountAmount decimal.Decimal,
) (*OrderRepository, error) {
	query := `
		INSERT INTO orders (user_id, status, promo_code_id, total_amount, discount_amount)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING *
	`
	var order OrderRepository
	err := tx.conn.GetContext(ctx, &order, query, userID, "CREATED", promoId, totalAmount, discountAmount)
	return &order, err
}

func (tx *OrdersTransaction) CreateOrderItems(
	ctx context.Context, orderID uuid.UUID, items *[]OrderItemToCreateRepository) (*[]OrderItemRepository, error) {
	if len(*items) == 0 {
		return nil, nil
	}

	query := "INSERT INTO order_items (order_id, product_id, quantity, price_at_order) VALUES "
	args := make([]any, 0, len(*items)*4)
	placeholders := make([]string, 0, len(*items))

	for i, item := range *items {
		base := i*4 + 1
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", base, base+1, base+2, base+3))
		args = append(args, orderID, item.ProductId, item.Quantity, item.PriceAtOrder)
	}

	query += strings.Join(placeholders, ", ")
	query += " RETURNING id, product_id, quantity, price_at_order"

	var result []OrderItemRepository
	err := tx.conn.SelectContext(ctx, &result, query, args...)
	return &result, err
}

func (tx *OrdersTransaction) CreateUserOperation(ctx context.Context, userID uuid.UUID, operationType string) error {
	query := "INSERT INTO user_operations (user_id, operation_type) VALUES ($1, $2)"
	_, err := tx.conn.ExecContext(ctx, query, userID, operationType)
	return err
}

func (tx *OrdersTransaction) GetForUpdateOrderById(ctx context.Context, id uuid.UUID) (*OrderRepository, error) {
	query := "SELECT * FROM orders WHERE id = $1 FOR UPDATE"
	var order OrderRepository
	err := tx.conn.GetContext(ctx, &order, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
	}
	return &order, err
}

func (tx *OrdersTransaction) UpdateOrderById(
	ctx context.Context, id uuid.UUID, promoCodeId *uuid.UUID, totalAmount, discountAmount decimal.Decimal,
) (*OrderRepository, error) {
	query := `
		UPDATE orders SET 
			promo_code_id = $1,
			total_amount = $2,
			discount_amount = $3
		WHERE id = $4
		RETURNING *
	`
	var order OrderRepository
	err := tx.conn.GetContext(ctx, &order, query, promoCodeId, totalAmount, discountAmount, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
	}
	return &order, err
}

func (tx *OrdersTransaction) ReturnProductsFromOrder(ctx context.Context, orderId uuid.UUID) error {
	query := `
		UPDATE products
		SET stock = stock + oi.quantity
		FROM order_items oi
		WHERE oi.order_id = $1 AND oi.product_id = products.id
	`
	_, err := tx.conn.ExecContext(ctx, query, orderId)
	return err
}

func (tx *OrdersTransaction) RemoveOrderItems(ctx context.Context, orderId uuid.UUID) error {
	query := "DELETE FROM order_items WHERE order_id = $1"
	_, err := tx.conn.ExecContext(ctx, query, orderId)
	return err
}

func (tx *OrdersTransaction) DecrementPromoUsers(ctx context.Context, promoId uuid.UUID) error {
	query := `
		UPDATE promo_codes
		SET current_uses = current_uses - 1
		WHERE id = $1
	`
	_, err := tx.conn.ExecContext(ctx, query, promoId)
	return err
}

func (tx *OrdersTransaction) UpdateOrderStatus(ctx context.Context, orderId uuid.UUID, status gen.OrderStatus) error {
	query := "UPDATE orders SET status = $1 WHERE id = $2"
	_, err := tx.conn.ExecContext(ctx, query, status, orderId)
	return err
}

func (tx *OrdersTransaction) GetForUpdatePromoCodeById(ctx context.Context, promoId uuid.UUID) (*PromoCodeRepository, error) {
	query := "SELECT * FROM promo_codes WHERE id = $1 FOR UPDATE"
	var promo PromoCodeRepository
	err := tx.conn.GetContext(ctx, &promo, query, promoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromoCodeNotFound
		}
	}
	return &promo, err
}
