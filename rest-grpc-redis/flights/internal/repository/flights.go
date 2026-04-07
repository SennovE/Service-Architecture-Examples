package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
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
	DepartureAirport   string    `db:"departure_airport"`
	DestinationAirport string    `db:"destination_airport"`
	DepartureDate      time.Time `db:"departure_date"`
	DestinationDate    time.Time `db:"destination_date"`
	TotalSeats         uint64    `db:"total_seats"`
	AvailableSeats     uint64    `db:"available_seats"`
	Price              decimal.Decimal
	Status             string
	Version            uint64
}

type FlightsRepository struct {
	conn *sqlx.DB
}

func NewFlightsRepository(conn *sqlx.DB) *FlightsRepository {
	return &FlightsRepository{conn: conn}
}

func (r *FlightsRepository) GetFlight(ctx context.Context, flightID uuid.UUID) (*Flight, error) {
	const query = "SELECT * FROM flights WHERE id = $1;"
	var flight Flight
	err := r.conn.GetContext(ctx, &flight, query, flightID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFlightNotFound
		}
		return nil, err
	}
	return &flight, nil
}

func (r *FlightsRepository) SearchFlights(
	ctx context.Context, origin, destination string, departureDate *time.Time) ([]Flight, error) {
	query := `
		SELECT * FROM flights
		WHERE status = 'SCHEDULED'
		  AND departure_airport = $1
   		  AND destination_airport = $2
	`
	var err error
	var flights []Flight
	if departureDate == nil {
		err = r.conn.SelectContext(ctx, &flights, query, origin, destination)
	} else {
		query += " AND departure_date = $3"
		err = r.conn.SelectContext(ctx, &flights, query, origin, destination, departureDate)
	}
	return flights, err
}

func (r *FlightsRepository) ReserveSeats(
	ctx context.Context, flightID, bookingID uuid.UUID, seats uint64) (decimal.Decimal, uint64, error) {
	const query0 = `
		SELECT available_seats FROM flights
		WHERE id = $1
		FOR UPDATE;
	`
	const query1 = `
		UPDATE flights
		SET available_seats = available_seats - $1,
		    version = version + 1
		WHERE id = $2
		RETURNING price, version;
	`
	const query2 = `
		INSERT INTO seat_reservations (flight_id, booking_id, seats, price, status)
		VALUES ($1, $2, $3, $4, $5);
	`
	const query3 = `
		SELECT price, seats FROM seat_reservations
		WHERE booking_id = $1;
	`
	tx, err := r.conn.BeginTxx(ctx, nil)
	if err != nil {
		return decimal.Zero, 0, err
	}
	defer tx.Rollback()

	var availableSeats int
	err = tx.GetContext(ctx, &availableSeats, query0, flightID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return decimal.Zero, 0, ErrFlightNotFound
		}
		return decimal.Zero, 0, err
	}
	if availableSeats < int(seats) {
		return decimal.Zero, 0, ErrNotEnoughSeats
	}

	var price decimal.Decimal
	var version uint64
	err = tx.QueryRowxContext(ctx, query1, seats, flightID).Scan(&price, &version)
	if err != nil {
		return decimal.Zero, 0, err
	}

	_, err = tx.ExecContext(ctx, query2, flightID, bookingID, int(seats), price, "ACTIVE")
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			type bookingInfo struct {
				Price decimal.Decimal
				Seats uint64
			}
			var info bookingInfo
			err = tx.GetContext(ctx, &info, query3, bookingID)
			if err != nil {
				return decimal.Zero, 0, err
			}
			if info.Seats == seats {
				return info.Price, 0, nil
			}
			return decimal.Zero, 0, ErrBookingExists
		}
		return decimal.Zero, 0, err
	}
	return price, version, tx.Commit()
}

func (r *FlightsRepository) ReleaseReservation(ctx context.Context, bookingID uuid.UUID) (uuid.UUID, uint64, error) {
	const query0 = `
		SELECT flight_id, seats, status
		FROM seat_reservations
		WHERE booking_id = $1
		FOR UPDATE;
	`
	const query1 = `
		UPDATE seat_reservations
		SET status = 'RELEASED'
		WHERE booking_id = $1;
	`
	const query2 = `
		UPDATE flights
		SET available_seats = available_seats + $1,
		    version = version + 1
		WHERE id = $2
		RETURNING version;
	`
	tx, err := r.conn.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, 0, err
	}
	defer tx.Rollback()

	type flightSeats struct {
		FlightID uuid.UUID `db:"flight_id"`
		Seats    int
		Status   string
	}
	var seats flightSeats
	err = tx.GetContext(ctx, &seats, query0, bookingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, 0, ErrBookingNotFound
		}
		return uuid.Nil, 0, err
	}
	if seats.Status == "RELEASED" {
		return uuid.Nil, 0, nil
	}

	_, err = tx.ExecContext(ctx, query1, bookingID)
	if err != nil {
		return uuid.Nil, 0, err
	}
	var version uint64
	err = tx.GetContext(ctx, &version, query2, seats.Seats, seats.FlightID)
	if err != nil {
		return uuid.Nil, 0, err
	}
	return seats.FlightID, version, tx.Commit()
}
