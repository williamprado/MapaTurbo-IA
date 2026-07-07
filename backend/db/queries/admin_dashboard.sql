-- name: GetAdminDashboardStats :one
SELECT 
    (SELECT COUNT(*) FROM organizations) as total_organizations,
    (SELECT COUNT(*) FROM organizations WHERE status = 'ACTIVE') as active_organizations,
    (SELECT COUNT(*) FROM users) as total_users,
    (SELECT COUNT(*) FROM users WHERE status = 'ACTIVE') as active_users,
    (SELECT COUNT(*) FROM mind_maps) as total_mind_maps,
    (SELECT COUNT(*) FROM uploads WHERE status != 'FAILED') as total_uploads,
    (SELECT COALESCE(SUM(amount), 0)::bigint FROM ai_credit_transactions WHERE type = 'SUB') as credits_consumed,
    (SELECT COUNT(*) FROM subscriptions WHERE status IN ('ACTIVE', 'TRIALING')) as active_subscriptions,
    (SELECT COALESCE(SUM(amount), 0)::numeric FROM invoices WHERE status = 'PAID') as paid_invoices_amount
;
