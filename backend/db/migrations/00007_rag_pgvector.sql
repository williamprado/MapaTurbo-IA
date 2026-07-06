-- +goose Up
-- 1. Document Sources
CREATE TABLE document_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE UNIQUE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'CHUNKING', -- CHUNKING, READY, FAILED
    word_count INTEGER NOT NULL DEFAULT 0,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Trigger for document_sources updated_at
CREATE TRIGGER tr_document_sources_updated_at
BEFORE UPDATE ON document_sources
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 2. Document Chunks
CREATE TABLE document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_source_id UUID NOT NULL REFERENCES document_sources(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    token_count INTEGER NOT NULL,
    embedding vector(1536) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_document_chunks_org ON document_chunks(organization_id);
CREATE INDEX idx_document_chunks_source ON document_chunks(document_source_id, chunk_index);

-- +goose Down
DROP TABLE IF EXISTS document_chunks CASCADE;
DROP TABLE IF EXISTS document_sources CASCADE;
