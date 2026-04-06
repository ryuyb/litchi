-- Add validation_config and detected_project columns to repositories table
-- These columns store execution validation configuration and project detection results

-- Add validation_config column (nullable JSONB)
ALTER TABLE repositories
ADD COLUMN IF NOT EXISTS validation_config JSONB;

-- Add detected_project column (nullable JSONB)
ALTER TABLE repositories
ADD COLUMN IF NOT EXISTS detected_project JSONB;

-- Add comments for documentation
COMMENT ON COLUMN repositories.validation_config IS 'Execution validation configuration (formatting, linting, testing settings)';
COMMENT ON COLUMN repositories.detected_project IS 'Detected project information (language, tools, confidence)';