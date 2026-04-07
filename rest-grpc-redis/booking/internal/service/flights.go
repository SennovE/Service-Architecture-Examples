package service

import (
	"booking/internal/repository"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrFlightNotFound = errors.New("flight not found")

type FlightsService struct {
	repo *repository.FlightsRepository
}

func NewFlightsService(repo *repository.FlightsRepository) *FlightsService {
	return &FlightsService{repo: repo}
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

func (s *FlightsService) GetFlights(ctx context.Context, origin, destination string, date *time.Time) ([]Flight, error) {
	flights, err := s.repo.GetFlights(ctx, origin, destination, date)
	if err != nil {
		if errors.Is(err, repository.ErrFlightNotFound) {
			return nil, ErrFlightNotFound
		}
		return nil, err
	}
	var flightsRes []Flight
	for _, flight := range flights {
		flightsRes = append(flightsRes, Flight(flight))
	}
	return flightsRes, nil
}

func (s *FlightsService) GetFlightsId(ctx context.Context, id uuid.UUID) (*Flight, error) {
	flight, err := s.repo.GetFlightsId(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrFlightNotFound) {
			return nil, ErrFlightNotFound
		}
		return nil, err
	}
	flightsRes := Flight(*flight)
	return &flightsRes, nil
}
