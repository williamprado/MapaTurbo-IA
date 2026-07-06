-- name: CreateGenerationJob :one
INSERT INTO generation_jobs (
    organization_id, user_id, type, status, input, credits_cost, started_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id;

-- name: GetGenerationJob :one
SELECT id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id
FROM generation_jobs
WHERE id = $1 LIMIT 1;

-- name: ListGenerationJobsByOrganization :many
SELECT id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id
FROM generation_jobs
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateGenerationJobStatus :one
UPDATE generation_jobs
SET status = $2,
    started_at = CASE WHEN $2 = 'PROCESSING' THEN NOW() ELSE started_at END,
    finished_at = CASE WHEN $2 = 'COMPLETED' OR $2 = 'FAILED' THEN NOW() ELSE finished_at END
WHERE id = $1
RETURNING id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id;

-- name: UpdateGenerationJobResult :one
UPDATE generation_jobs
SET status = 'COMPLETED',
    result = $2,
    mind_map_id = $3,
    finished_at = NOW()
WHERE id = $1
RETURNING id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id;

-- name: UpdateGenerationJobError :one
UPDATE generation_jobs
SET status = 'FAILED',
    error = $2,
    finished_at = NOW()
WHERE id = $1
RETURNING id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id;

-- name: CreateMindMap :one
INSERT INTO mind_maps (
    organization_id, user_id, title, source_type, source_upload_id, status, json_data, is_public
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, organization_id, user_id, title, source_type, source_upload_id, status, json_data, is_public, created_at, updated_at;

-- name: GetMindMap :one
SELECT id, organization_id, user_id, title, source_type, source_upload_id, status, json_data, is_public, created_at, updated_at
FROM mind_maps
WHERE id = $1 LIMIT 1;

-- name: ListMindMapsByOrganization :many
SELECT id, organization_id, user_id, title, source_type, source_upload_id, status, json_data, is_public, created_at, updated_at
FROM mind_maps
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountMindMapsByOrganization :one
SELECT COUNT(*) FROM mind_maps WHERE organization_id = $1;

-- name: UpdateMindMapData :one
UPDATE mind_maps
SET title = COALESCE($2, title),
    json_data = COALESCE($3, json_data),
    is_public = COALESCE($4, is_public),
    status = COALESCE($5, status),
    updated_at = NOW()
WHERE id = $1
RETURNING id, organization_id, user_id, title, source_type, source_upload_id, status, json_data, is_public, created_at, updated_at;

-- name: DeleteMindMap :exec
DELETE FROM mind_maps WHERE id = $1;
