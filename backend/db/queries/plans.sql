-- name: CreatePlan :one
INSERT INTO plans (
    name, description, price_monthly, price_yearly, currency,
    credits_monthly, max_maps, max_files, max_users, max_storage_bytes,
    features, is_public, is_active, target_organization_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, name, description, price_monthly, price_yearly, currency, credits_monthly, max_maps, max_files, max_users, max_storage_bytes, features, is_public, is_active, target_organization_id, created_at, updated_at;

-- name: GetPlanByID :one
SELECT id, name, description, price_monthly, price_yearly, currency, credits_monthly, max_maps, max_files, max_users, max_storage_bytes, features, is_public, is_active, target_organization_id, created_at, updated_at
FROM plans
WHERE id = $1 LIMIT 1;

-- name: UpdatePlan :one
UPDATE plans
SET name = COALESCE($2, name),
    description = COALESCE($3, description),
    price_monthly = COALESCE($4, price_monthly),
    price_yearly = COALESCE($5, price_yearly),
    currency = COALESCE($6, currency),
    credits_monthly = COALESCE($7, credits_monthly),
    max_maps = COALESCE($8, max_maps),
    max_files = COALESCE($9, max_files),
    max_users = COALESCE($10, max_users),
    max_storage_bytes = COALESCE($11, max_storage_bytes),
    features = COALESCE($12, features),
    is_public = COALESCE($13, is_public),
    is_active = COALESCE($14, is_active),
    target_organization_id = COALESCE($15, target_organization_id)
WHERE id = $1
RETURNING id, name, description, price_monthly, price_yearly, currency, credits_monthly, max_maps, max_files, max_users, max_storage_bytes, features, is_public, is_active, target_organization_id, created_at, updated_at;

-- name: ListPlans :many
SELECT id, name, price_monthly, price_yearly, currency, is_public, is_active, target_organization_id, created_at
FROM plans
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPlans :one
SELECT COUNT(*) FROM plans;

-- name: ListPublicPlans :many
SELECT id, name, description, price_monthly, price_yearly, currency, credits_monthly, max_maps, max_files, max_users, max_storage_bytes, features, is_public, is_active, created_at
FROM plans
WHERE is_public = TRUE AND is_active = TRUE
ORDER BY price_monthly ASC;

-- name: CreateSubscription :one
INSERT INTO subscriptions (
    organization_id, plan_id, status, payment_provider,
    external_subscription_id, current_period_start, current_period_end,
    trial_start, trial_end
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, organization_id, plan_id, status, payment_provider, external_subscription_id, current_period_start, current_period_end, trial_start, trial_end, created_at, updated_at;

-- name: GetSubscriptionByOrg :one
SELECT id, organization_id, plan_id, status, payment_provider, external_subscription_id, current_period_start, current_period_end, trial_start, trial_end, created_at, updated_at
FROM subscriptions
WHERE organization_id = $1 LIMIT 1;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET plan_id = COALESCE($2, plan_id),
    status = COALESCE($3, status),
    payment_provider = COALESCE($4, payment_provider),
    external_subscription_id = COALESCE($5, external_subscription_id),
    current_period_start = COALESCE($6, current_period_start),
    current_period_end = COALESCE($7, current_period_end),
    trial_start = COALESCE($8, trial_start),
    trial_end = COALESCE($9, trial_end)
WHERE id = $1
RETURNING id, organization_id, plan_id, status, payment_provider, external_subscription_id, current_period_start, current_period_end, trial_start, trial_end, created_at, updated_at;
