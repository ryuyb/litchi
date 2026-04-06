-- Add installation_id column for GitHub App authentication support
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS installation_id BIGINT DEFAULT 0;

-- Create index for faster lookup by installation_id
CREATE INDEX IF NOT EXISTS idx_repositories_installation_id ON repositories(installation_id);

-- Add comment for documentation
COMMENT ON COLUMN repositories.installation_id IS 'GitHub App Installation ID (0 if not installed via App)';