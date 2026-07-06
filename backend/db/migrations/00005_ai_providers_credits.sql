-- +goose Up
-- 1. AI Providers
CREATE TABLE ai_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL, -- openai, gemini, grok
    api_key_secure TEXT NOT NULL, -- Encrypted
    base_url VARCHAR(255),
    default_model VARCHAR(100) NOT NULL,
    text_model VARCHAR(100) NOT NULL,
    vision_model VARCHAR(100) NOT NULL,
    audio_model VARCHAR(100) NOT NULL,
    embedding_model VARCHAR(100) NOT NULL,
    embedding_dimensions INTEGER NOT NULL DEFAULT 1536,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 1,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    limit_per_minute INTEGER NOT NULL DEFAULT 0,
    limit_per_day INTEGER NOT NULL DEFAULT 0,
    cost_per_credit DECIMAL(10, 4) NOT NULL DEFAULT 1.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Trigger for ai_providers updated_at
CREATE TRIGGER tr_ai_providers_updated_at
BEFORE UPDATE ON ai_providers
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 2. AI Credit Balances
CREATE TABLE ai_credit_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE UNIQUE,
    balance INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Trigger for ai_credit_balances updated_at
CREATE TRIGGER tr_ai_credit_balances_updated_at
BEFORE UPDATE ON ai_credit_balances
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 3. AI Credit Transactions
CREATE TABLE ai_credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    amount INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL, -- CREDIT, DEBIT
    description VARCHAR(255) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_ai_credit_transactions_org_created ON ai_credit_transactions(organization_id, created_at DESC);

-- 4. AI Action Prices
CREATE TABLE ai_action_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_key VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    credits_cost INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Trigger for ai_action_prices updated_at
CREATE TRIGGER tr_ai_action_prices_updated_at
BEFORE UPDATE ON ai_action_prices
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS ai_action_prices CASCADE;
DROP TABLE IF EXISTS ai_credit_transactions CASCADE;
DROP TABLE IF EXISTS ai_credit_balances CASCADE;
DROP TABLE IF EXISTS ai_providers CASCADE;
