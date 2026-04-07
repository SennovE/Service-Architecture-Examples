CREATE TYPE order_type AS ENUM (
    'CREATED',
    'PAYMENT_PENDING',
    'PAID',
    'SHIPPED',
    'COMPLETED',
    'CANCELED'
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    user_id UUID NOT NULL,
    status order_type NOT NULL,
    promo_code_id UUID REFERENCES promo_codes (id),
    total_amount DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0
);

CREATE TRIGGER trg_orders_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
