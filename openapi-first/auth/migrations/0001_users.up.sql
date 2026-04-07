CREATE TYPE user_type AS ENUM ('USER', 'SELLER', 'ADMIN');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE,
    password_hash TEXT,
    role user_type DEFAULT 'USER'
);
