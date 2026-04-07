package service

import (
	"booking/internal/repository"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrBookingNotFound = errors.New("booking not found")
	ErrNotEnoughSeats  = errors.New("not enough seats")
	ErrBookingExists   = errors.New("booking already exists")
)

type BookingService struct {
	repo *repository.BookingRepository
}

func NewBookingService(repo *repository.BookingRepository) *BookingService {
	return &BookingService{repo: repo}
}

type BookingToCreate struct {
	UserID         uuid.UUID
	FlightID       uuid.UUID
	PassengerName  string
	PassengerEmail string
	Seats          uint64
}

type Booking struct {
	BookingToCreate
	ID        uuid.UUID
	CreatedAt time.Time
	Price     decimal.Decimal
	Status    string
}

func (s *BookingService) GetBookings(ctx context.Context, userID uuid.UUID) ([]Booking, error) {
	bookings, err := s.repo.GetBookings(ctx, userID)
	if err != nil {
		return nil, err
	}
	bookingsRes := make([]Booking, len(bookings))
	for i, booking := range bookings {
		bookingsRes[i] = mapBookingRepositoryToService(booking)
	}
	return bookingsRes, nil
}

func (s *BookingService) PostBookings(ctx context.Context, bookingToCreat BookingToCreate) (*Booking, error) {
	booking, err := s.repo.PostBookings(ctx, repository.BookingToCreate{
		UserID:         bookingToCreat.UserID,
		FlightID:       bookingToCreat.FlightID,
		PassengerName:  bookingToCreat.PassengerName,
		PassengerEmail: bookingToCreat.PassengerEmail,
		Seats:          bookingToCreat.Seats,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrFlightNotFound):
			return nil, ErrFlightNotFound
		case errors.Is(err, repository.ErrNotEnoughSeats):
			return nil, ErrNotEnoughSeats
		case errors.Is(err, repository.ErrBookingExists):
			return nil, ErrBookingExists
		default:
			return nil, err
		}
	}
	bookingRes := mapBookingRepositoryToService(*booking)
	return &bookingRes, nil
}

func (s *BookingService) GetBookingsId(ctx context.Context, id uuid.UUID) (*Booking, error) {
	booking, err := s.repo.GetBookingsId(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrBookingNotFound) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	bookingRes := mapBookingRepositoryToService(*booking)
	return &bookingRes, nil
}

func (s *BookingService) PostBookingsIdCancel(ctx context.Context, id uuid.UUID) error {
	err := s.repo.PostBookingsIdCancel(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrBookingNotFound) {
			return ErrBookingNotFound
		}
		return err
	}
	return nil
}

func mapBookingRepositoryToService(booking repository.Booking) Booking {
	return Booking{
		BookingToCreate: BookingToCreate{
			FlightID:       booking.FlightID,
			PassengerName:  booking.PassengerName,
			PassengerEmail: booking.PassengerEmail,
			UserID:         booking.UserID,
			Seats:          booking.Seats,
		},
		ID:        booking.ID,
		CreatedAt: booking.CreatedAt,
		Price:     booking.Price,
		Status:    booking.Status,
	}
}
