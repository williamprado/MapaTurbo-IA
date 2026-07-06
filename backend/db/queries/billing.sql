-- name: CreatePaymentCustomer :one
INSERT INTO payment_customers (
    organization_id, provider, external_customer_id, name, email, document, phone, payload
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, organization_id, provider, external_customer_id, name, email, document, phone, payload, created_at, updated_at;

-- name: GetPaymentCustomer :one
SELECT id, organization_id, provider, external_customer_id, name, email, document, phone, payload, created_at, updated_at
FROM payment_customers
WHERE provider = $1 AND organization_id = $2 LIMIT 1;

-- name: GetPaymentCustomerByExternalID :one
SELECT id, organization_id, provider, external_customer_id, name, email, document, phone, payload, created_at, updated_at
FROM payment_customers
WHERE provider = $1 AND external_customer_id = $2 LIMIT 1;

-- name: CreatePaymentProvider :one
INSERT INTO payment_providers (
    name, slug, api_key_secure, webhook_secret_secure, is_active, mode
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, slug, api_key_secure, webhook_secret_secure, is_active, mode, created_at;

-- name: GetPaymentProviderBySlug :one
SELECT id, name, slug, api_key_secure, webhook_secret_secure, is_active, mode, created_at
FROM payment_providers
WHERE slug = $1 LIMIT 1;

-- name: ListPaymentProviders :many
SELECT id, name, slug, api_key_secure, webhook_secret_secure, is_active, mode, created_at
FROM payment_providers
ORDER BY name ASC;

-- name: UpdatePaymentProvider :one
UPDATE payment_providers
SET api_key_secure = COALESCE($2, api_key_secure),
    webhook_secret_secure = COALESCE($3, webhook_secret_secure),
    is_active = COALESCE($4, is_active),
    mode = COALESCE($5, mode)
WHERE id = $1
RETURNING id, name, slug, api_key_secure, webhook_secret_secure, is_active, mode, created_at;

-- name: CreateInvoice :one
INSERT INTO invoices (
    organization_id, subscription_id, amount, currency, status, external_invoice_id, pdf_url, due_date, billing_type, invoice_url, bank_slip_url, pix_qr_code, pix_copy_paste
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id, organization_id, subscription_id, amount, currency, status, external_invoice_id, pdf_url, due_date, billing_type, invoice_url, bank_slip_url, pix_qr_code, pix_copy_paste, paid_at, created_at, updated_at;

-- name: GetInvoiceByID :one
SELECT id, organization_id, subscription_id, amount, currency, status, external_invoice_id, pdf_url, due_date, billing_type, invoice_url, bank_slip_url, pix_qr_code, pix_copy_paste, paid_at, created_at, updated_at
FROM invoices
WHERE id = $1 LIMIT 1;

-- name: GetInvoiceByExternalID :one
SELECT id, organization_id, subscription_id, amount, currency, status, external_invoice_id, pdf_url, due_date, billing_type, invoice_url, bank_slip_url, pix_qr_code, pix_copy_paste, paid_at, created_at, updated_at
FROM invoices
WHERE external_invoice_id = $1 LIMIT 1;

-- name: UpdateInvoiceStatus :one
UPDATE invoices
SET status = $2,
    paid_at = CASE WHEN $2 = 'PAID' THEN NOW() ELSE paid_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING id, organization_id, subscription_id, amount, currency, status, external_invoice_id, pdf_url, due_date, billing_type, invoice_url, bank_slip_url, pix_qr_code, pix_copy_paste, paid_at, created_at, updated_at;

-- name: UpdateInvoiceDetails :one
UPDATE invoices
SET status = COALESCE($2, status),
    pdf_url = COALESCE($3, pdf_url),
    due_date = COALESCE($4, due_date),
    billing_type = COALESCE($5, billing_type),
    invoice_url = COALESCE($6, invoice_url),
    bank_slip_url = COALESCE($7, bank_slip_url),
    pix_qr_code = COALESCE($8, pix_qr_code),
    pix_copy_paste = COALESCE($9, pix_copy_paste),
    paid_at = COALESCE($10, paid_at),
    updated_at = NOW()
WHERE id = $1
RETURNING id, organization_id, subscription_id, amount, currency, status, external_invoice_id, pdf_url, due_date, billing_type, invoice_url, bank_slip_url, pix_qr_code, pix_copy_paste, paid_at, created_at, updated_at;

-- name: ListInvoicesByOrganization :many
SELECT id, organization_id, subscription_id, amount, currency, status, external_invoice_id, pdf_url, due_date, billing_type, invoice_url, bank_slip_url, pix_qr_code, pix_copy_paste, paid_at, created_at
FROM invoices
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountInvoicesByOrganization :one
SELECT COUNT(*) FROM invoices WHERE organization_id = $1;

-- name: ListInvoicesDetailed :many
SELECT i.id, i.organization_id, i.subscription_id, i.amount, i.currency, i.status, i.external_invoice_id, i.due_date, i.billing_type, i.invoice_url, i.paid_at, i.created_at,
       o.name as organization_name
FROM invoices i
JOIN organizations o ON i.organization_id = o.id
ORDER BY i.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountInvoices :one
SELECT COUNT(*) FROM invoices;

-- name: CreatePaymentTransaction :one
INSERT INTO payment_transactions (
    organization_id, invoice_id, amount, provider, external_transaction_id, status, payment_method, payload
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, organization_id, invoice_id, amount, provider, external_transaction_id, status, payment_method, payload, created_at;

-- name: GetPaymentTransactionByExternalID :one
SELECT id, organization_id, invoice_id, amount, provider, external_transaction_id, status, payment_method, payload, created_at
FROM payment_transactions
WHERE external_transaction_id = $1 LIMIT 1;

-- name: UpdatePaymentTransactionStatus :one
UPDATE payment_transactions
SET status = $2,
    paid_at = CASE WHEN $2 = 'PAID' THEN NOW() ELSE paid_at END,
    failed_at = CASE WHEN $2 = 'FAILED' THEN NOW() ELSE failed_at END
WHERE id = $1
RETURNING id, organization_id, invoice_id, amount, provider, external_transaction_id, status, payment_method, payload, created_at;

-- name: ListPaymentTransactions :many
SELECT t.id, t.organization_id, t.invoice_id, t.amount, t.provider, t.external_transaction_id, t.status, t.payment_method, t.created_at,
       o.name as organization_name
FROM payment_transactions t
JOIN organizations o ON t.organization_id = o.id
ORDER BY t.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPaymentTransactions :one
SELECT COUNT(*) FROM payment_transactions;

-- name: CreateWebhookEvent :one
INSERT INTO webhook_events (
    provider, event_type, external_id, payload, status, error, processed_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, provider, event_type, external_id, payload, status, error, processed_at, created_at;

-- name: GetWebhookEventByExternalID :one
SELECT id, provider, event_type, external_id, payload, status, error, processed_at, created_at
FROM webhook_events
WHERE provider = $1 AND external_id = $2 LIMIT 1;

-- name: UpdateWebhookEventStatus :one
UPDATE webhook_events
SET status = $2,
    error = COALESCE($3, error),
    processed_at = CASE WHEN $2 = 'PROCESSED' OR $2 = 'FAILED' THEN NOW() ELSE processed_at END
WHERE id = $1
RETURNING id, provider, event_type, external_id, payload, status, error, processed_at, created_at;

-- name: ListWebhookEvents :many
SELECT id, provider, event_type, external_id, status, error, processed_at, created_at
FROM webhook_events
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountWebhookEvents :one
SELECT COUNT(*) FROM webhook_events;
