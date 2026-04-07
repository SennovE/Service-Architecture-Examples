package repository

import (
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

type ProductsRepository struct {
	conn *sqlx.DB
}

func NewProductsRepository(db *sqlx.DB) *ProductsRepository {
	return &ProductsRepository{conn: db}
}

var ErrProductNotFound = errors.New("Product not found")

type ProductPaginationRepository struct {
	Size     int
	Offset   int
	Status   *gen.ProductStatus
	Category *string
}

type ProductToCreateRepository struct {
	SellerId    uuid.UUID `db:"seller_id"`
	Name        string
	Description *string
	Price       decimal.Decimal
	Stock       int
	Category    string
	Status      gen.ProductStatus
}

type ProductRepository struct {
	Id        uuid.UUID
	CreatedAt *time.Time `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
	ProductToCreateRepository
}

func (db *ProductsRepository) CreateProduct(
	ctx context.Context, newProduct ProductToCreateRepository) (*ProductRepository, error) {
	query, err := db.conn.PrepareNamed(`
		INSERT INTO products (name, description, price, stock, category, status, seller_id)
		VALUES (:name, :description, :price, :stock, :category, :status, :seller_id)
		RETURNING *
	`)
	if err != nil {
		return nil, err
	}
	defer query.Close()
	var product ProductRepository
	err = query.GetContext(ctx, &product, newProduct)
	return &product, err
}

func (db *ProductsRepository) GetProductById(ctx context.Context, id uuid.UUID) (*ProductRepository, error) {
	query := "SELECT * FROM products WHERE id = $1"
	var product ProductRepository
	err := db.conn.GetContext(ctx, &product, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
	}
	return &product, err
}

func (db *ProductsRepository) GetProductsWithPagination(
	ctx context.Context, pagination ProductPaginationRepository) (*[]ProductRepository, error) {
	whereParts := []string(nil)
	if pagination.Category != nil {
		whereParts = append(whereParts, "category = :category")
	}
	if pagination.Status != nil {
		whereParts = append(whereParts, "status = :status")
	}
	stmt := `
		SELECT * FROM products %s
		ORDER BY created_at
		LIMIT :size
		OFFSET :offset
	`
	if len(whereParts) != 0 {
		part := fmt.Sprintf("WHERE %s", strings.Join(whereParts, " AND "))
		stmt = fmt.Sprintf(stmt, part)
	} else {
		stmt = fmt.Sprintf(stmt, "")
	}
	query, err := db.conn.PrepareNamed(stmt)
	if err != nil {
		return nil, err
	}
	defer query.Close()
	var products []ProductRepository
	err = query.SelectContext(ctx, &products, pagination)
	return &products, err
}

func (db *ProductsRepository) GetTotalWithPagination(
	ctx context.Context, pagination ProductPaginationRepository) (int, error) {
	whereParts := []string(nil)
	if pagination.Category != nil {
		whereParts = append(whereParts, "category = :category")
	}
	if pagination.Status != nil {
		whereParts = append(whereParts, "status = :status")
	}
	stmt := "SELECT COUNT(1) as total FROM products %s"
	if len(whereParts) != 0 {
		part := fmt.Sprintf("WHERE %s", strings.Join(whereParts, " AND "))
		stmt = fmt.Sprintf(stmt, part)
	} else {
		stmt = fmt.Sprintf(stmt, "")
	}
	query, err := db.conn.PrepareNamed(stmt)
	if err != nil {
		return 0, err
	}
	defer query.Close()
	type TotalAmount struct {
		Total int
	}
	var total TotalAmount
	err = query.GetContext(ctx, &total, pagination)
	return total.Total, err
}

func (db *ProductsRepository) UpdateProduct(
	ctx context.Context, id uuid.UUID, newProduct ProductToCreateRepository) (*ProductRepository, error) {
	setParts := []string{
		"name = :name",
		"price = :price",
		"stock = :stock",
		"category = :category",
		"status = :status",
	}
	if newProduct.Description != nil {
		setParts = append(setParts, "description = :description")
	}
	stmt := fmt.Sprintf("UPDATE products SET %s WHERE id = :id RETURNING *", strings.Join(setParts, ", "))
	query, err := db.conn.PrepareNamed(stmt)
	if err != nil {
		return nil, err
	}
	defer query.Close()
	type ProductToUpdateRepository struct {
		Id uuid.UUID
		ProductToCreateRepository
	}
	values := ProductToUpdateRepository{id, newProduct}
	var product ProductRepository
	err = query.GetContext(ctx, &product, values)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
	}
	return &product, err
}

func (db *ProductsRepository) ArchiveProduct(ctx context.Context, id uuid.UUID) (*ProductRepository, error) {
	query := `
		UPDATE products SET status = 'ARCHIVED'
		WHERE id = $1
		RETURNING *
	`
	var product ProductRepository
	err := db.conn.GetContext(ctx, &product, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
	}
	return &product, err
}
