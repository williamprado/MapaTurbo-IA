-- name: CreateDocumentSource :one
INSERT INTO document_sources (
    upload_id, organization_id, title, status, word_count, chunk_count
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, upload_id, organization_id, title, status, word_count, chunk_count, created_at, updated_at;

-- name: GetDocumentSourceByUpload :one
SELECT id, upload_id, organization_id, title, status, word_count, chunk_count, created_at, updated_at
FROM document_sources
WHERE upload_id = $1 LIMIT 1;

-- name: UpdateDocumentSourceStatus :one
UPDATE document_sources
SET status = $2, chunk_count = $3, word_count = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, status, chunk_count, word_count, updated_at;

-- name: CreateDocumentChunk :one
INSERT INTO document_chunks (
    document_source_id, organization_id, content, chunk_index, token_count, embedding, metadata
)
VALUES ($1, $2, $3, $4, $5, sqlc.arg(embedding)::vector, $6)
RETURNING id, document_source_id, organization_id, content, chunk_index, token_count, created_at;

-- name: SearchSimilarChunks :many
SELECT c.id, c.content, c.chunk_index, c.document_source_id,
       (1 - (c.embedding <=> sqlc.arg(query_embedding)::vector))::float8 AS similarity
FROM document_chunks c
JOIN document_sources s ON s.id = c.document_source_id
WHERE c.organization_id = $1
  AND s.upload_id = $2
ORDER BY c.embedding <=> sqlc.arg(query_embedding)::vector
LIMIT $3;

-- name: GetDocumentChunksBySource :many
SELECT id, content, chunk_index, token_count
FROM document_chunks
WHERE document_source_id = $1
ORDER BY chunk_index ASC
LIMIT $2;
