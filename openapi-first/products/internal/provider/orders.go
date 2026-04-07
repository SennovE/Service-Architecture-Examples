package provider

import (
	"net/http"
	"products/internal/gen"
	"products/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (prov *Provider) CreateOrder(ctx *gin.Context) {
	userID, _ := ctx.Get("userID")
	var req gen.CreateOrderJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	items := make([]service.OrderItemToCreateService, len(req.Items))
	for i, item := range req.Items {
		items[i] = service.OrderItemToCreateService{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		}
	}
	values := service.OrderToCreateService{
		Items:     items,
		PromoCode: req.PromoCode,
	}
	order, err := prov.ordersSrvc.CreateOrder(ctx, values, userID.(uuid.UUID))

	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusCreated,
		mapOrderServiceToProvider(order),
	)
}

func (prov *Provider) GetOrderById(ctx *gin.Context, id gen.IdPathParam) {
	userID, _ := ctx.Get("userID")
	role := ctx.GetString("role")

	order, err := prov.ordersSrvc.GetOrderById(ctx, id, userID.(uuid.UUID), role)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusOK,
		mapOrderServiceToProvider(order),
	)
}

func (prov *Provider) UpdateOrder(ctx *gin.Context, id gen.IdPathParam) {
	userID, _ := ctx.Get("userID")
	role := ctx.GetString("role")
	var req gen.UpdateOrderJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	items := make([]service.OrderItemToCreateService, len(req.Items))
	for i, item := range req.Items {
		items[i] = service.OrderItemToCreateService{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		}
	}
	order, err := prov.ordersSrvc.UpdateOrderItems(ctx, id, &items, userID.(uuid.UUID), role)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusOK,
		mapOrderServiceToProvider(order),
	)
}

func (prov *Provider) CancelOrder(ctx *gin.Context, id gen.IdPathParam) {
	userID, _ := ctx.Get("userID")
	role := ctx.GetString("role")

	err := prov.ordersSrvc.CancelOrder(ctx, id, userID.(uuid.UUID), role)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func mapOrderServiceToProvider(order *service.OrderService) *gen.OrderResponse {
	mappedItems := make([]gen.OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		mappedItems[i] = gen.OrderItemResponse{
			Id:           item.Id,
			PriceAtOrder: item.PriceAtOrder,
			ProductId:    item.ProductId,
			Quantity:     item.Quantity,
		}
	}
	return &gen.OrderResponse{
		CreatedAt:      order.CreatedAt,
		DiscountAmount: order.DiscountAmount,
		Id:             &order.Id,
		PromoCodeId:    order.PromoCodeId,
		Status:         order.Status,
		TotalAmount:    order.TotalAmount,
		UpdatedAt:      order.UpdatedAt,
		UserId:         &order.UserID,
		Items:          mappedItems,
	}
}
