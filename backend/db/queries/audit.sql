-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    actor_user_id, organization_id, action, entity_type, entity_id, metadata, ip, user_agent
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, actor_user_id, organization_id, action, entity_type, entity_id, metadata, ip, user_agent, created_at;

-- name: ListAuditLogs :many
SELECT id, actor_user_id, organization_id, action, entity_type, entity_id, metadata, ip, user_agent, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;
