CREATE TYPE booking_status AS ENUM ('CONFIRMED', 'CANCELLED');

CREATE TABLE booking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT now(),
    flight_id UUID NOT NULL,
    user_id UUID NOT NULL,
    passenger_name TEXT NOT NULL,
    passenger_email TEXT NOT NULL,
    seats INTEGER NOT NULL,
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    status booking_status NOT NULL
);
