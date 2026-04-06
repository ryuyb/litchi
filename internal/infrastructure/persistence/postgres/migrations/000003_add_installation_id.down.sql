-- Remove installation_id column
DROP INDEX IF EXISTS idx_repositories_installation_id;
ALTER TABLE repositories DROP COLUMN IF EXISTS installation_id;