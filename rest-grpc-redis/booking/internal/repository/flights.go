package repository

import (
	"booking/internal/gen/proto"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrFlightNotFound = errors.New("flight not found")

type FlightsRepository struct {
	client proto.FlightsClient
}

func NewFlightsRepository(client proto.FlightsClient) *FlightsRepository {
	return &FlightsRepository{
		client: client,
	}
}

type Flight struct {
	ID                 uuid.UUID
	Num                string
	Company            string
	IATA               string
	DepartureAirport   string
	DestinationAirport string
	DepartureDate      time.Time
	DestinationDate    time.Time
	TotalSeats         uint64
	AvailableSeats     uint64
	Price              decimal.Decimal
	Status             string
}

func (r *FlightsRepository) GetFlights(
	ctx context.Context, origin, destination string, date *time.Time) ([]Flight, error) {
	req := &proto.SearchFlightsRequest{}
	var t *timestamppb.Timestamp
	if date != nil {
		t = timestamppb.New(*date)
	}
	req.SetOrigin(origin)
	req.SetDestination(destination)
	req.SetDepartureDate(t)
	stream, err := r.client.SearchFlights(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC error: %v", err)
	}
	var flights []Flight
	for {
		flight, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("gRPC error: %v", err)
		}
		flights = append(flights, mapProtoToRepository(flight))
	}
	return flights, nil
}

func (r *FlightsRepository) GetFlightsId(ctx context.Context, id uuid.UUID) (*Flight, error) {
	req := &proto.GetFlightRequest{}
	req.SetFlightId(id.String())
	resp, err := r.client.GetFlight(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return nil, fmt.Errorf("gRPC error: %v", err)
		}

		switch st.Code() {
		case codes.NotFound:
			return nil, ErrFlightNotFound
		default:
			return nil, fmt.Errorf("gRPC error: %v", err)
		}
	}
	flight := mapProtoToRepository(resp)
	return &flight, nil
}

func mapProtoToRepository(flight *proto.FlightResponse) Flight {
	price, _ := decimal.NewFromString(flight.GetPrice())
	return Flight{
		ID:                 uuid.MustParse(flight.GetId()),
		Num:                flight.GetNum(),
		Company:            flight.GetCompany(),
		IATA:               flight.GetIata(),
		DepartureAirport:   flight.GetDepartureAirport(),
		DestinationAirport: flight.GetDestinationAirport(),
		DepartureDate:      flight.GetDepartureDate().AsTime(),
		DestinationDate:    flight.GetDestinationDate().AsTime(),
		TotalSeats:         flight.GetTotalSeats(),
		AvailableSeats:     flight.GetAvailableSeats(),
		Price:              price,
		Status:             proto.FlightStatus_name[int32(flight.GetStatus())],
	}
}
