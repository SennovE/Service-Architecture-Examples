CREATE TYPE promo_code_discount_type AS ENUM ('PERCENTAGE', 'FIXED_AMOUNT');

CREATE TABLE promo_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) UNIQUE NOT NULL,
    discount_type promo_code_discount_type NOT NULL,
    discount_value DECIMAL(12,2) NOT NULL,
    min_order_amount DECIMAL(12,2) NOT NULL,
    max_uses INTEGER NOT NULL,
    current_uses INTEGER DEFAULT 0,
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    active BOOLEAN DEFAULT true
);
