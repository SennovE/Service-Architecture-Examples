package service

import (
	"context"
	"errors"
	"flights/internal/repository"
	"iter"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFlightNotFound  = errors.New("flight not found")
	ErrBookingNotFound = errors.New("booking not found")
	ErrNotEnoughSeats  = errors.New("not enough seats")
	ErrBookingExists   = errors.New("booking already exists")
)

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
	Version            uint64
}

type FlightsService struct {
	db *repository.FlightsRepository
}

func NewFlightsService(db *repository.FlightsRepository) *FlightsService {
	return &FlightsService{db: db}
}

func (s *FlightsService) GetFlight(
	ctx context.Context, flightID uuid.UUID) (*Flight, error) {
	flight, err := s.db.GetFlight(ctx, flightID)
	if err != nil {
		if errors.Is(err, repository.ErrFlightNotFound) {
			return nil, ErrFlightNotFound
		}
		return nil, err
	}
	return mapRepositoryToService(flight), nil
}

func (s *FlightsService) SearchFlights(
	ctx context.Context, origin, destination string, departureDate *time.Time) (iter.Seq[*Flight], error) {
	flights, err := s.db.SearchFlights(ctx, origin, destination, departureDate)
	if err != nil {
		return nil, err
	}
	return func(yield func(*Flight) bool) {
		for _, flight := range flights {
			if !yield(mapRepositoryToService(&flight)) {
				return
			}
		}
	}, nil
}

func (s *FlightsService) ReleaseReservation(ctx context.Context, bookingID uuid.UUID) (uuid.UUID, uint64, error) {
	flightID, version, err := s.db.ReleaseReservation(ctx, bookingID)
	if err != nil {
		if errors.Is(err, repository.ErrBookingNotFound) {
			return uuid.Nil, 0, ErrBookingNotFound
		}
		return uuid.Nil, 0, err
	}
	return flightID, version, nil
}

func (s *FlightsService) ReserveSeats(
	ctx context.Context, flightID, bookingID uuid.UUID, seats uint64) (decimal.Decimal, uint64, error) {
	price, version, err := s.db.ReserveSeats(ctx, flightID, bookingID, seats)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrFlightNotFound):
			return decimal.Zero, 0, ErrFlightNotFound
		case errors.Is(err, repository.ErrNotEnoughSeats):
			return decimal.Zero, 0, ErrNotEnoughSeats
		case errors.Is(err, repository.ErrBookingExists):
			return decimal.Zero, 0, ErrBookingExists
		default:
			return decimal.Zero, 0, err
		}
	}
	return price, version, nil
}

func mapRepositoryToService(flight *repository.Flight) *Flight {
	return &Flight{
		ID:                 flight.ID,
		Num:                flight.Num,
		Company:            flight.Company,
		IATA:               flight.IATA,
		DepartureAirport:   flight.DepartureAirport,
		DestinationAirport: flight.DestinationAirport,
		DepartureDate:      flight.DepartureDate,
		DestinationDate:    flight.DestinationDate,
		TotalSeats:         flight.TotalSeats,
		AvailableSeats:     flight.AvailableSeats,
		Price:              flight.Price,
		Status:             flight.Status,
		Version:            flight.Version,
	}
}
