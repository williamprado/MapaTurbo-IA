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

-- name: ListAuditLogsDetailed :many
SELECT al.id, al.actor_user_id, al.organization_id, al.action, al.entity_type, al.entity_id, al.metadata, al.ip, al.user_agent, al.created_at,
       u.email as actor_email, o.name as organization_name
FROM audit_logs al
LEFT JOIN users u ON al.actor_user_id = u.id
LEFT JOIN organizations o ON al.organization_id = o.id
ORDER BY al.created_at DESC
LIMIT $1 OFFSET $2;
