package provider

import (
	"net/http"
	"products/internal/gen"
	"products/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (prov *Provider) CreateProduct(ctx *gin.Context) {
	userID, _ := ctx.Get("userID")
	var req gen.CreateProductJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	values := service.ProductToCreateService{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Status:      req.Status,
	}

	product, err := prov.productsSrvc.CreateProduct(ctx, values, userID.(uuid.UUID))
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusCreated,
		mapProductServiceToProvider(product),
	)
}

func (prov *Provider) GetProductById(ctx *gin.Context, id gen.IdPathParam) {
	product, err := prov.productsSrvc.GetProductById(ctx, id)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusOK,
		mapProductServiceToProvider(product),
	)
}

func (prov *Provider) ListProducts(ctx *gin.Context, params gen.ListProductsParams) {
	values := service.ProductPaginationService{
		Size:     20,
		Page:     0,
		Status:   params.Status,
		Category: params.Category,
	}
	if params.Size != nil {
		values.Size = *params.Size
	}
	if params.Page != nil {
		values.Page = *params.Page
	}

	products, err := prov.productsSrvc.GetProductsWithPagination(ctx, values)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	resp := &gen.ProductsPageResponse{
		Items:         make([]gen.ProductResponse, products.Size),
		Page:          products.Page,
		Size:          products.Size,
		TotalElements: products.TotalElements,
	}
	for i, item := range products.Items {
		resp.Items[i] = *mapProductServiceToProvider(&item)
	}
	ctx.JSON(http.StatusOK, resp)
}

func (prov *Provider) ArchiveProduct(ctx *gin.Context, id gen.IdPathParam) {
	userID, _ := ctx.Get("userID")
	userRole := ctx.GetString("role")

	product, err := prov.productsSrvc.ArchiveProduct(ctx, id, userID.(uuid.UUID), userRole)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusOK,
		mapProductServiceToProvider(product),
	)
}

func (prov *Provider) UpdateProduct(ctx *gin.Context, id gen.IdPathParam) {
	userID, _ := ctx.Get("userID")
	userRole := ctx.GetString("role")
	var req gen.UpdateProductJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	values := service.ProductToCreateService{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Status:      req.Status,
	}

	product, err := prov.productsSrvc.UpdateProduct(ctx, id, values, userID.(uuid.UUID), userRole)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusOK,
		mapProductServiceToProvider(product),
	)
}

func mapProductServiceToProvider(product *service.ProductService) *gen.ProductResponse {
	return &gen.ProductResponse{
		Id:          &product.Id,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
		SellerId:    &product.SellerId,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		Category:    product.Category,
		Status:      product.Status,
	}
}
