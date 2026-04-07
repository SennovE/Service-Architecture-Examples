package provider

import (
	"booking/internal/gen/api"
	"booking/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FlightsProvider struct {
	srvc *service.FlightsService
}

func NewFlightsProvider(srvc *service.FlightsService) *FlightsProvider {
	return &FlightsProvider{srvc: srvc}
}

func (p *FlightsProvider) GetFlights(ctx *gin.Context, params api.GetFlightsParams) {
	var date *time.Time
	if params.Date != nil {
		date = &params.Date.Time
	}
	flights, err := p.srvc.GetFlights(ctx, params.Origin, params.Destination, date)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}
	resp := make([]api.Flight, len(flights))
	for i, flight := range flights {
		resp[i] = mapFlightsServiceToProvider(&flight)
	}
	ctx.JSON(
		http.StatusOK,
		map[string][]api.Flight{"flights": resp},
	)
}

func (p *FlightsProvider) GetFlightsId(ctx *gin.Context, id uuid.UUID) {
	flight, err := p.srvc.GetFlightsId(ctx, id)
	if err != nil {
		makeErrorResponse(ctx, err)
		return
	}
	ctx.JSON(
		http.StatusOK,
		mapFlightsServiceToProvider(flight),
	)
}

func mapFlightsServiceToProvider(flight *service.Flight) api.Flight {
	return api.Flight{
		Id:                 flight.ID,
		Num:                flight.Num,
		Company:            flight.Company,
		Iata:               flight.IATA,
		DepartureAirport:   flight.DepartureAirport,
		DestinationAirport: flight.DestinationAirport,
		DepartureDate:      flight.DepartureDate,
		DestinationDate:    flight.DestinationDate,
		TotalSeats:         flight.TotalSeats,
		AvailableSeats:     flight.AvailableSeats,
		Price:              flight.Price.String(),
		Status:             api.FlightStatus(flight.Status),
	}
}
