package repository

import (
	"booking/internal/gen/proto"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrBookingNotFound = errors.New("booking not found")
	ErrNotEnoughSeats  = errors.New("not enough seats")
	ErrBookingExists   = errors.New("booking already exists")
)

type BookingRepository struct {
	dbConn     *sqlx.DB
	grpcClient proto.FlightsClient
}

func NewBookingRepository(dbConn *sqlx.DB, grpcClient proto.FlightsClient) *BookingRepository {
	return &BookingRepository{dbConn: dbConn, grpcClient: grpcClient}
}

type BookingToCreate struct {
	UserID         uuid.UUID `db:"user_id"`
	FlightID       uuid.UUID `db:"flight_id"`
	PassengerName  string    `db:"passenger_name"`
	PassengerEmail string    `db:"passenger_email"`
	Seats          uint64
}

type Booking struct {
	BookingToCreate
	ID        uuid.UUID
	CreatedAt time.Time `db:"created_at"`
	Price     decimal.Decimal
	Status    string
}

func (r *BookingRepository) GetBookings(ctx context.Context, userID uuid.UUID) ([]Booking, error) {
	const query = "SELECT * FROM booking WHERE user_id = $1"
	var bookings []Booking
	err := r.dbConn.SelectContext(ctx, &bookings, query, userID)
	if err != nil {
		return nil, err
	}
	return bookings, nil
}

func (r *BookingRepository) GetBookingsId(ctx context.Context, id uuid.UUID) (*Booking, error) {
	const query = "SELECT * FROM booking WHERE id = $1"
	var bookings Booking
	err := r.dbConn.GetContext(ctx, &bookings, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	return &bookings, nil
}

func (r *BookingRepository) PostBookingsIdCancel(ctx context.Context, id uuid.UUID) error {
	const query1 = "SELECT status FROM booking WHERE id = $1 FOR UPDATE;"
	const query2 = "UPDATE booking SET status = $1 WHERE id = $2;"
	tx, err := r.dbConn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	err = tx.GetContext(ctx, &status, query1, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBookingNotFound
		}
		return err
	}
	if status == "CANCELLED" {
		return nil
	}
	_, err = tx.ExecContext(ctx, query2, "CANCELLED", id)
	if err != nil {
		return err
	}
	err = r.releaseReservation(ctx, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *BookingRepository) PostBookings(ctx context.Context, booking BookingToCreate) (*Booking, error) {
	bookingID := uuid.New()
	price, err := r.reserveSeats(ctx, booking.FlightID, bookingID, booking.Seats)
	if err != nil {
		return nil, err
	}
	query := `
		INSERT INTO booking (
			id,
			user_id,
			flight_id,
			passenger_name,
			passenger_email,
			seats,
			price,
			status
		) VALUES (
		 	$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING *;
	`
	tx, err := r.dbConn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var bookingRes Booking
	err = tx.GetContext(
		ctx,
		&bookingRes,
		query,
		bookingID,
		booking.UserID,
		booking.FlightID,
		booking.PassengerName,
		booking.PassengerEmail,
		booking.Seats,
		price.Mul(decimal.NewFromUint64(booking.Seats)),
		"CONFIRMED",
	)
	if err != nil {
		return nil, err
	}
	return &bookingRes, tx.Commit()
}

func (r *BookingRepository) releaseReservation(ctx context.Context, id uuid.UUID) error {
	req := &proto.ReleaseReservationRequest{}
	req.SetBookingId(id.String())
	_, err := r.grpcClient.ReleaseReservation(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return fmt.Errorf("gRPC error: %v", err)
		}

		switch st.Code() {
		case codes.NotFound:
			return ErrBookingNotFound
		}
		return fmt.Errorf("gRPC error: %v", err)
	}
	return err
}

func (r *BookingRepository) reserveSeats(ctx context.Context, flightID, bookingID uuid.UUID, seats uint64) (price decimal.Decimal, err error) {
	req := &proto.ReserveSeatsRequest{}
	req.SetFlightId(flightID.String())
	req.SetBookingId(bookingID.String())
	req.SetSeatCount(seats)
	resp, err := r.grpcClient.ReserveSeats(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return decimal.Zero, fmt.Errorf("gRPC error: %v", err)
		}

		switch st.Code() {
		case codes.NotFound:
			return decimal.Zero, ErrFlightNotFound
		case codes.ResourceExhausted:
			return decimal.Zero, ErrNotEnoughSeats
		case codes.AlreadyExists:
			return decimal.Zero, ErrBookingExists
		}
		return decimal.Zero, fmt.Errorf("gRPC error: %v", err)
	}
	price, err = decimal.NewFromString(resp.GetPrice())
	return price, err
}
