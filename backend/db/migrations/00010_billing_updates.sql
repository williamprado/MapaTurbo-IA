-- +goose Up
-- 1. Create payment_customers
CREATE TABLE payment_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    external_customer_id VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    email VARCHAR(255),
    document VARCHAR(50),
    phone VARCHAR(50),
    payload JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT uq_payment_customer_provider_org UNIQUE (provider, organization_id)
);
CREATE INDEX idx_payment_customers_org ON payment_customers(organization_id);
CREATE INDEX idx_payment_customers_external ON payment_customers(external_customer_id);

-- Trigger for payment_customers updated_at
CREATE TRIGGER tr_payment_customers_updated_at
BEFORE UPDATE ON payment_customers
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 2. Update subscriptions
ALTER TABLE subscriptions
ADD COLUMN external_customer_id VARCHAR(255),
ADD COLUMN external_checkout_id VARCHAR(255),
ADD COLUMN cancel_at_period_end BOOLEAN DEFAULT FALSE,
ADD COLUMN canceled_at TIMESTAMP WITH TIME ZONE;

-- 3. Update invoices
ALTER TABLE invoices
ADD COLUMN due_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN billing_type VARCHAR(50),
ADD COLUMN invoice_url TEXT,
ADD COLUMN bank_slip_url TEXT,
ADD COLUMN pix_qr_code TEXT,
ADD COLUMN pix_copy_paste TEXT;

-- 4. Update payment_transactions
ALTER TABLE payment_transactions
ADD COLUMN payment_method VARCHAR(50),
ADD COLUMN paid_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN failed_at TIMESTAMP WITH TIME ZONE;

-- 5. Update payment_providers
ALTER TABLE payment_providers
ADD COLUMN mode VARCHAR(50) NOT NULL DEFAULT 'sandbox';

-- +goose Down
ALTER TABLE payment_providers DROP COLUMN IF EXISTS mode;

ALTER TABLE payment_transactions
DROP COLUMN IF EXISTS payment_method,
DROP COLUMN IF EXISTS paid_at,
DROP COLUMN IF EXISTS failed_at;

ALTER TABLE invoices
DROP COLUMN IF EXISTS due_date,
DROP COLUMN IF EXISTS billing_type,
DROP COLUMN IF EXISTS invoice_url,
DROP COLUMN IF EXISTS bank_slip_url,
DROP COLUMN IF EXISTS pix_qr_code,
DROP COLUMN IF EXISTS pix_copy_paste;

ALTER TABLE subscriptions
DROP COLUMN IF EXISTS external_customer_id,
DROP COLUMN IF EXISTS external_checkout_id,
DROP COLUMN IF EXISTS cancel_at_period_end,
DROP COLUMN IF EXISTS canceled_at;

DROP TABLE IF EXISTS payment_customers CASCADE;
