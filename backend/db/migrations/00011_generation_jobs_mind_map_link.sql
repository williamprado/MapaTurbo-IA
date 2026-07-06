-- +goose Up
-- Add mind_map_id to generation_jobs
ALTER TABLE generation_jobs
ADD COLUMN mind_map_id UUID REFERENCES mind_maps(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE generation_jobs
DROP COLUMN IF EXISTS mind_map_id;
