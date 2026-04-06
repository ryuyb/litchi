-- Rollback: Remove validation_config and detected_project columns from repositories table

ALTER TABLE repositories
DROP COLUMN IF EXISTS detected_project;

ALTER TABLE repositories
DROP COLUMN IF EXISTS validation_config;