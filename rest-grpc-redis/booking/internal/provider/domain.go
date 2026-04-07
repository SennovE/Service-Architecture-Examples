package provider

import (
	"booking/internal/gen/api"
	"booking/internal/gen/proto"
	"booking/internal/repository"
	"booking/internal/service"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type Provider struct {
	*FlightsProvider
	*BookingProvider
}

func NewProvider(dbConn *sqlx.DB, grpcClient proto.FlightsClient) api.ServerInterface {
	return &Provider{
		FlightsProvider: NewFlightsProvider(
			service.NewFlightsService(
				repository.NewFlightsRepository(grpcClient),
			),
		),
		BookingProvider: NewBookingProvider(
			service.NewBookingService(
				repository.NewBookingRepository(dbConn, grpcClient),
			),
		),
	}
}

func validateRequestBody[T any](ctx *gin.Context, req *T) bool {
	if err := ctx.ShouldBindJSON(req); err != nil {
		sendErrorResponse(
			ctx, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(),
		)
		return false
	}
	return true
}

func makeErrorResponse(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrFlightNotFound):
		sendErrorResponse(ctx, http.StatusNotFound, "FLIGHT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrBookingNotFound):
		sendErrorResponse(ctx, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrNotEnoughSeats):
		sendErrorResponse(ctx, http.StatusConflict, "NOT_ENOUGH_SEATS", err.Error())
	case errors.Is(err, service.ErrBookingExists):
		sendErrorResponse(ctx, http.StatusConflict, "BOOKING_EXISTS", err.Error())
	default:
		sendErrorResponse(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func sendErrorResponse(ctx *gin.Context, statusCode int, errorCode string, msg string) {
	if strings.HasPrefix(msg, "gRPC error:") {
		ctx.Set("GRPC_ERROR", struct{}{})
	}
	errResp := api.ErrorResponse{
		ErrorCode: errorCode,
		Message:   msg,
	}
	ctx.JSON(
		statusCode,
		errResp,
	)
}
