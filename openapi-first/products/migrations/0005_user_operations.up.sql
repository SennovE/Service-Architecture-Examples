CREATE TYPE user_operation_type AS ENUM ('CREATE_ORDER', 'UPDATE_ORDER');

CREATE TABLE user_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT now(),
    user_id UUID NOT NULL,
    operation_type user_operation_type NOT NULL
);
