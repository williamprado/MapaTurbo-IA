-- name: CreateUpload :one
INSERT INTO uploads (
    organization_id, user_id, filename, original_name, mime_type, size, storage_provider, storage_key, status, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, organization_id, user_id, filename, original_name, mime_type, size, storage_provider, storage_key, status, metadata, created_at, updated_at;

-- name: GetUploadByID :one
SELECT id, organization_id, user_id, filename, original_name, mime_type, size, storage_provider, storage_key, status, metadata, created_at, updated_at
FROM uploads
WHERE id = $1 LIMIT 1;

-- name: ListUploadsByOrganization :many
SELECT id, organization_id, user_id, filename, original_name, mime_type, size, storage_provider, storage_key, status, metadata, created_at
FROM uploads
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUploadsByOrganization :one
SELECT COUNT(*) FROM uploads WHERE organization_id = $1 AND status != 'FAILED';

-- name: SumUploadSizeByOrganization :one
SELECT COALESCE(SUM(size), 0)::bigint FROM uploads WHERE organization_id = $1 AND status != 'FAILED';

-- name: UpdateUploadStatus :one
UPDATE uploads
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, status, updated_at;

