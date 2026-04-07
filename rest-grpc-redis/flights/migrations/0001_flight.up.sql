CREATE TYPE flight_status AS ENUM ('SCHEDULED', 'DEPARTED', 'CANCELLED', 'COMPLETED');

CREATE TABLE flights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    num TEXT NOT NULL,
    company TEXT NOT NULL,
    iata TEXT NOT NULL,
    departure_airport TEXT NOT NULL,
    destination_airport TEXT NOT NULL,
    departure_date TIMESTAMP NOT NULL,
    destination_date TIMESTAMP NOT NULL,
    total_seats INTEGER NOT NULL CHECK (total_seats > 0),
    available_seats INTEGER NOT NULL CHECK (available_seats >= 0 AND available_seats <= total_seats),
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    status flight_status NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,

    CONSTRAINT flights_num_departure_date_uniq UNIQUE (num, departure_date)
);
