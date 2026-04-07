CREATE TYPE reservation_status AS ENUM ('ACTIVE', 'RELEASED', 'EXPIRED');

CREATE TABLE seat_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flight_id UUID NOT NULL REFERENCES flights (id),
    booking_id UUID NOT NULL UNIQUE,
    seats INTEGER NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    status reservation_status NOT NULL
);
