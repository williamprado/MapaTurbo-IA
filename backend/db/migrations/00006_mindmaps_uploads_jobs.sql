-- +goose Up
-- 1. Uploads
CREATE TABLE uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    storage_provider VARCHAR(50) NOT NULL DEFAULT 'MINIO',
    storage_key VARCHAR(555) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'UPLOADED', -- UPLOADED, PROCESSING, PROCESSED, FAILED
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_uploads_org_status ON uploads(organization_id, status);

-- Trigger for uploads updated_at
CREATE TRIGGER tr_uploads_updated_at
BEFORE UPDATE ON uploads
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 2. Mind Maps
CREATE TABLE mind_maps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL, -- TEXT, PDF, URL, YOUTUBE, TOPIC
    source_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'GENERATING', -- GENERATING, READY, FAILED
    json_data JSONB NOT NULL DEFAULT '{}',
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_mindmaps_org ON mind_maps(organization_id);

-- Trigger for mind_maps updated_at
CREATE TRIGGER tr_mind_maps_updated_at
BEFORE UPDATE ON mind_maps
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 3. Generation Jobs
CREATE TABLE generation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    input JSONB,
    result JSONB,
    error TEXT,
    credits_cost INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_generation_jobs_org_status ON generation_jobs(organization_id, status);

-- +goose Down
DROP TABLE IF EXISTS generation_jobs CASCADE;
DROP TABLE IF EXISTS mind_maps CASCADE;
DROP TABLE IF EXISTS uploads CASCADE;
