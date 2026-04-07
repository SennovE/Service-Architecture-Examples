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

type ProductsService struct {
	db *repository.ProductsRepository
}

func NewProductsService(db *repository.ProductsRepository) *ProductsService {
	return &ProductsService{db: db}
}

type ProductPaginationService struct {
	Size     int
	Page     int
	Status   *gen.ProductStatus
	Category *string
}

type ProductToCreateService struct {
	Name        string
	Description *string
	Price       decimal.Decimal
	Stock       int
	Category    string
	Status      gen.ProductStatus
}

type ProductService struct {
	Id          uuid.UUID
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	SellerId    uuid.UUID
	Name        string
	Description *string
	Price       decimal.Decimal
	Stock       int
	Category    string
	Status      gen.ProductStatus
}

type ProductsListService struct {
	Items         []ProductService
	TotalElements int
	Size          int
	Page          int
}

func (srvc *ProductsService) CreateProduct(
	ctx context.Context, newProduct ProductToCreateService, sellerId uuid.UUID) (*ProductService, error) {
	values := repository.ProductToCreateRepository{
		SellerId:    sellerId,
		Name:        newProduct.Name,
		Description: newProduct.Description,
		Price:       newProduct.Price,
		Stock:       newProduct.Stock,
		Category:    newProduct.Category,
		Status:      newProduct.Status,
	}
	product, err := srvc.db.CreateProduct(ctx, values)
	if err != nil {
		return nil, err
	}
	return mapProductRepositoryToService(product), nil
}

func (srvc *ProductsService) GetProductById(ctx context.Context, id uuid.UUID) (*ProductService, error) {
	product, err := srvc.db.GetProductById(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, &ErrProductNotFound{IDs: []uuid.UUID{id}}
		}
		return nil, err
	}
	return mapProductRepositoryToService(product), nil
}

func (srvc *ProductsService) GetProductsWithPagination(
	ctx context.Context, pagination ProductPaginationService) (*ProductsListService, error) {
	paginationRepo := repository.ProductPaginationRepository{
		Size:     pagination.Size,
		Offset:   pagination.Size * pagination.Page,
		Status:   pagination.Status,
		Category: pagination.Category,
	}
	items, err := srvc.db.GetProductsWithPagination(ctx, paginationRepo)
	if err != nil {
		return nil, err
	}
	total, err := srvc.db.GetTotalWithPagination(ctx, paginationRepo)
	if err != nil {
		return nil, err
	}
	products := ProductsListService{
		Items:         make([]ProductService, len(*items)),
		TotalElements: total,
		Size:          len(*items),
		Page:          pagination.Page,
	}
	for i, item := range *items {
		products.Items[i] = *mapProductRepositoryToService(&item)
	}
	return &products, nil
}

func (srvc *ProductsService) UpdateProduct(
	ctx context.Context, id uuid.UUID, newProduct ProductToCreateService, sellerId uuid.UUID, role string,
) (*ProductService, error) {
	if err := srvc.checkPermission(ctx, id, sellerId, role); err != nil {
		return nil, err
	}
	values := repository.ProductToCreateRepository{
		SellerId:    sellerId,
		Name:        newProduct.Name,
		Description: newProduct.Description,
		Price:       newProduct.Price,
		Stock:       newProduct.Stock,
		Category:    newProduct.Category,
		Status:      newProduct.Status,
	}
	product, err := srvc.db.UpdateProduct(ctx, id, repository.ProductToCreateRepository(values))
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, &ErrProductNotFound{IDs: []uuid.UUID{id}}
		}
		return nil, err
	}
	return mapProductRepositoryToService(product), nil
}

func (srvc *ProductsService) ArchiveProduct(
	ctx context.Context, id uuid.UUID, sellerId uuid.UUID, role string) (*ProductService, error) {
	if err := srvc.checkPermission(ctx, id, sellerId, role); err != nil {
		return nil, err
	}
	product, err := srvc.db.ArchiveProduct(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, &ErrProductNotFound{IDs: []uuid.UUID{id}}
		}
		return nil, err
	}
	return mapProductRepositoryToService(product), nil
}

func (srvc *ProductsService) checkPermission(
	ctx context.Context, productId uuid.UUID, sellerId uuid.UUID, role string) error {
	if role != "ADMIN" {
		product, err := srvc.db.GetProductById(ctx, productId)
		if err != nil {
			if errors.Is(err, repository.ErrProductNotFound) {
				return &ErrProductNotFound{IDs: []uuid.UUID{productId}}
			}
			return err
		}
		if product.SellerId != sellerId {
			return ErrNoPermission
		}
	}
	return nil
}

func mapProductRepositoryToService(product *repository.ProductRepository) *ProductService {
	return &ProductService{
		Id:          product.Id,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
		SellerId:    product.SellerId,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		Category:    product.Category,
		Status:      product.Status,
	}
}
