package provider

import (
	"errors"
	"net/http"
	"products/internal/gen"
	"products/internal/service"

	"github.com/gin-gonic/gin"
)

type Provider struct {
	productsSrvc   *service.ProductsService
	promosCodeSrvc *service.PromoCodesService
	ordersSrvc     *service.OrdersService
}

func NewProvider(
	productsSrvc *service.ProductsService,
	promosCodeSrvc *service.PromoCodesService,
	ordersSrvc *service.OrdersService,
) *Provider {
	return &Provider{
		productsSrvc:   productsSrvc,
		promosCodeSrvc: promosCodeSrvc,
		ordersSrvc:     ordersSrvc,
	}
}

func validateRequestBody[T any](ctx *gin.Context, req *T) bool {
	if err := ctx.ShouldBindJSON(req); err != nil {
		sendErrorResponse(
			ctx, http.StatusBadRequest, gen.VALIDATIONERROR, "invalid body", err.Error(),
		)
		return false
	}
	return true
}

func makeErrorResponse(ctx *gin.Context, err error) {
	if _, ok := err.(*service.ErrOrderLimit); ok {
		sendErrorResponse(
			ctx,
			http.StatusTooManyRequests,
			gen.ORDERLIMITEXCEEDED,
			"order creation frequency limit has been exceeded",
			err.Error(),
		)
		return
	} else if _, ok := err.(*service.ErrHasActiveOrdeer); ok {
		sendErrorResponse(
			ctx,
			http.StatusConflict,
			gen.ORDERHASACTIVE,
			"user already has an active order",
			err.Error(),
		)
	} else if _, ok := err.(*service.ErrProductNotFound); ok {
		sendErrorResponse(
			ctx,
			http.StatusNotFound,
			gen.PRODUCTNOTFOUND,
			"product(s) not found",
			err.Error(),
		)
	} else if _, ok := err.(*service.ErrProductNotActive); ok {
		sendErrorResponse(
			ctx,
			http.StatusConflict,
			gen.PRODUCTINACTIVE,
			"product(s) not active",
			err.Error(),
		)
	} else if _, ok := err.(*service.ErrProductNotEnough); ok {
		sendErrorResponse(
			ctx,
			http.StatusConflict,
			gen.INSUFFICIENTSTOCK,
			"product(s) not enough",
			err.Error(),
		)
	} else if errors.Is(err, service.ErrPromoCodeInvalid) {
		sendErrorResponse(
			ctx,
			http.StatusUnprocessableEntity,
			gen.PROMOCODEINVALID,
			err.Error(),
			"",
		)
	} else if errors.Is(err, service.ErrPromoCodeMinAmount) {
		sendErrorResponse(
			ctx,
			http.StatusUnprocessableEntity,
			gen.PROMOCODEMINAMOUNT,
			err.Error(),
			"",
		)
	} else if errors.Is(err, service.ErrOrderNotFound) {
		sendErrorResponse(
			ctx,
			http.StatusNotFound,
			gen.ORDERNOTFOUND,
			"order not found",
			"",
		)
	} else if errors.Is(err, service.ErrNoPermission) {
		sendErrorResponse(
			ctx,
			http.StatusForbidden,
			gen.ORDEROWNERSHIPVIOLATION,
			"not enough permissions",
			"",
		)
	} else if errors.Is(err, service.ErrSamePromoExists) {
		sendErrorResponse(
			ctx,
			http.StatusConflict,
			"DUPLICATED PROMOCODE",
			"duplicated promocode",
			"promocode with same code exists",
		)
		return
	} else if errors.Is(err, service.ErrInvalidStateTransaction) {
		sendErrorResponse(
			ctx,
			http.StatusConflict,
			gen.INVALIDSTATETRANSITION,
			"invalid state transition",
			err.Error(),
		)
		return
	} else {
		sendErrorResponse(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", err.Error())
	}
}

func sendErrorResponse(ctx *gin.Context, statusCode int, errorCode gen.ErrorCode, msg, err string) {
	errResp := gen.ErrorResponse{
		ErrorCode: errorCode,
		Message:   msg,
	}
	if err != "" {
		errResp.Details = &map[string]any{"error": err}
	}
	ctx.JSON(
		statusCode,
		errResp,
	)
}
