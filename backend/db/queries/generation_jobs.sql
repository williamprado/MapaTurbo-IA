-- name: ListGenerationJobsByOrganizationPaginated :many
SELECT id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id
FROM generation_jobs
WHERE organization_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountGenerationJobsByOrganization :one
SELECT COUNT(*) FROM generation_jobs
WHERE organization_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'));
