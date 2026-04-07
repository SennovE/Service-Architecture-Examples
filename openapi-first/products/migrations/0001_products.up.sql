CREATE TYPE product_status AS ENUM ('ACTIVE', 'INACTIVE', 'ARCHIVED');

CREATE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    name VARCHAR(255) NOT NULL,
    description VARCHAR(4000),
    price DECIMAL(12,2) NOT NULL,
    stock INTEGER NOT NULL,
    category VARCHAR(100) NOT NULL,
    status product_status NOT NULL,
    seller_id UUID NOT NULL
);

CREATE INDEX idx_status ON products (status);

CREATE TRIGGER trg_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
