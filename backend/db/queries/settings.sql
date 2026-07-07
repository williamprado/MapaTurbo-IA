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

-- name: ListAiActionPrices :many
SELECT id, action_key, name, description, credits_cost, is_active, metadata, created_at, updated_at
FROM ai_action_prices
ORDER BY action_key ASC;

-- name: UpdateAiActionPrice :one
UPDATE ai_action_prices
SET credits_cost = $2,
    is_active = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, action_key, name, description, credits_cost, is_active, metadata, created_at, updated_at;

-- name: ListAiProviders :many
SELECT id, name, slug, api_key_secure, base_url, default_model, text_model, vision_model, audio_model, embedding_model, embedding_dimensions, is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit, created_at, updated_at
FROM ai_providers
ORDER BY priority DESC, name ASC;

-- name: UpdateAiProvider :one
UPDATE ai_providers
SET name = COALESCE($2, name),
    api_key_secure = COALESCE($3, api_key_secure),
    base_url = COALESCE($4, base_url),
    default_model = COALESCE($5, default_model),
    text_model = COALESCE($6, text_model),
    vision_model = COALESCE($7, vision_model),
    audio_model = COALESCE($8, audio_model),
    embedding_model = COALESCE($9, embedding_model),
    embedding_dimensions = COALESCE($10, embedding_dimensions),
    is_active = COALESCE($11, is_active),
    priority = COALESCE($12, priority),
    is_default = COALESCE($13, is_default),
    limit_per_minute = COALESCE($14, limit_per_minute),
    limit_per_day = COALESCE($15, limit_per_day),
    cost_per_credit = COALESCE($16, cost_per_credit),
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, slug, api_key_secure, base_url, default_model, text_model, vision_model, audio_model, embedding_model, embedding_dimensions, is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit, created_at, updated_at;

-- name: SetAllAiProvidersNotDefault :exec
UPDATE ai_providers
SET is_default = FALSE;

-- name: SetAiProviderDefault :exec
UPDATE ai_providers
SET is_default = TRUE
WHERE id = $1;

-- name: GetAiProviderByID :one
SELECT id, name, slug, api_key_secure, base_url, default_model, text_model, vision_model, audio_model, embedding_model, embedding_dimensions, is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit, created_at, updated_at
FROM ai_providers
WHERE id = $1 LIMIT 1;

-- name: ListCreditTransactionsByOrganization :many
SELECT id, organization_id, amount, type, description, metadata, created_at
FROM ai_credit_transactions
WHERE organization_id = $1
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountCreditTransactionsByOrganization :one
SELECT COUNT(*) FROM ai_credit_transactions
WHERE organization_id = $1
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'));
