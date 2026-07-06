-- name: GetSystemSetting :one
SELECT key, value, description, is_public, updated_at
FROM system_settings
WHERE key = $1 LIMIT 1;

-- name: UpsertSystemSetting :one
INSERT INTO system_settings (key, value, description, is_public)
VALUES ($1, $2, $3, $4)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = COALESCE(EXCLUDED.description, system_settings.description),
    is_public = EXCLUDED.is_public,
    updated_at = NOW()
RETURNING key, value, description, is_public, updated_at;

-- name: ListSystemSettings :many
SELECT key, value, description, is_public, updated_at
FROM system_settings
ORDER BY key ASC;

-- name: CreateAiProvider :one
INSERT INTO ai_providers (
    name, slug, api_key_secure, base_url, default_model, text_model,
    vision_model, audio_model, embedding_model, embedding_dimensions,
    is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING id, name, slug, api_key_secure, base_url, default_model, text_model, vision_model, audio_model, embedding_model, embedding_dimensions, is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit, created_at, updated_at;

-- name: GetAiProviderBySlug :one
SELECT id, name, slug, api_key_secure, base_url, default_model, text_model, vision_model, audio_model, embedding_model, embedding_dimensions, is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit, created_at, updated_at
FROM ai_providers
WHERE slug = $1 LIMIT 1;

-- name: GetDefaultAiProvider :one
SELECT id, name, slug, api_key_secure, base_url, default_model, text_model, vision_model, audio_model, embedding_model, embedding_dimensions, is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit
FROM ai_providers
WHERE is_active = TRUE AND is_default = TRUE LIMIT 1;

-- name: GetAiActionPrice :one
SELECT id, action_key, name, description, credits_cost, is_active, metadata, created_at, updated_at
FROM ai_action_prices
WHERE action_key = $1 LIMIT 1;

-- name: CreateAiActionPrice :one
INSERT INTO ai_action_prices (action_key, name, description, credits_cost, is_active, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, action_key, name, description, credits_cost, is_active, metadata, created_at, updated_at;

-- name: InitializeCreditBalance :one
INSERT INTO ai_credit_balances (organization_id, balance)
VALUES ($1, $2)
ON CONFLICT (organization_id) DO NOTHING
RETURNING id, organization_id, balance, updated_at;

-- name: GetCreditBalance :one
SELECT id, organization_id, balance, updated_at
FROM ai_credit_balances
WHERE organization_id = $1 LIMIT 1;

-- name: UpdateCreditBalance :one
UPDATE ai_credit_balances
SET balance = balance + $2
WHERE organization_id = $1
RETURNING id, organization_id, balance, updated_at;

-- name: CreateCreditTransaction :one
INSERT INTO ai_credit_transactions (organization_id, amount, type, description, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, organization_id, amount, type, description, metadata, created_at;
