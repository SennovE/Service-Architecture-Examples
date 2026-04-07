package provider

import (
	"net/http"
	"products/internal/gen"
	"products/internal/service"

	"github.com/gin-gonic/gin"
)

func (prov *Provider) CreatePromoCode(ctx *gin.Context) {
	var req gen.CreatePromoCodeJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	promCode, err := prov.promosCodeSrvc.CreatePromoCode(ctx, service.PromoCodeToCreateService(req))
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusCreated,
		&gen.PromoCodeResponse{
			Active:         promCode.Active,
			Code:           promCode.Code,
			CurrentUses:    &promCode.CurrentUses,
			DiscountType:   promCode.DiscountType,
			DiscountValue:  promCode.DiscountValue,
			Id:             &promCode.Id,
			MaxUses:        promCode.MaxUses,
			MinOrderAmount: promCode.MinOrderAmount,
			ValidFrom:      promCode.ValidFrom,
			ValidUntil:     promCode.ValidUntil,
		},
	)
}
