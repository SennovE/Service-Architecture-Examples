package provider

import (
	"booking/internal/gen/api"
	"booking/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
)

type BookingProvider struct {
	srvc *service.BookingService
}

func NewBookingProvider(srvc *service.BookingService) *BookingProvider {
	return &BookingProvider{srvc: srvc}
}

func (p *BookingProvider) GetBookings(ctx *gin.Context, params api.GetBookingsParams) {
	bookings, err := p.srvc.GetBookings(ctx, params.UserId)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}
	resp := make([]api.Booking, len(bookings))
	for i, booking := range bookings {
		resp[i] = mapBookingServiceToProvider(booking)
	}
	ctx.JSON(
		http.StatusOK,
		map[string][]api.Booking{"bookings": resp},
	)
}

func (p *BookingProvider) GetBookingsId(ctx *gin.Context, id uuid.UUID) {
	booking, err := p.srvc.GetBookingsId(ctx, id)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}
	ctx.JSON(
		http.StatusOK,
		mapBookingServiceToProvider(*booking),
	)
}

func (p *BookingProvider) PostBookings(ctx *gin.Context) {
	var req api.CreateBookingRequest
	if !validateRequestBody(ctx, &req) {
		return
	}
	booking, err := p.srvc.PostBookings(ctx, service.BookingToCreate{
		FlightID:       req.FlightId,
		UserID:         req.UserId,
		PassengerEmail: string(req.PassengerEmail),
		PassengerName:  req.PassengerName,
		Seats:          req.SeatCount,
	})
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}
	ctx.JSON(
		http.StatusCreated,
		mapBookingServiceToProvider(*booking),
	)
}

func (p *BookingProvider) PostBookingsIdCancel(ctx *gin.Context, id uuid.UUID) {
	err := p.srvc.PostBookingsIdCancel(ctx, id)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func mapBookingServiceToProvider(booking service.Booking) api.Booking {
	return api.Booking{
		Id:             booking.ID,
		CreatedAt:      booking.CreatedAt,
		FlightId:       booking.FlightID,
		UserId:         booking.UserID,
		PassengerEmail: types.Email(booking.PassengerEmail),
		PassengerName:  booking.PassengerName,
		Seats:          booking.Seats,
		TotalPrice:     booking.Price.Mul(decimal.NewFromUint64(booking.Seats)).String(),
		Status:         api.BookingStatus(booking.Status),
	}
}
