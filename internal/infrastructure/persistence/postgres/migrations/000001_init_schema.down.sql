-- T1.2.1: Rollback initial database schema
-- Drops all tables in reverse order of creation

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_executions_updated_at ON executions;
DROP TRIGGER IF EXISTS trigger_tasks_updated_at ON tasks;
DROP TRIGGER IF EXISTS trigger_designs_updated_at ON designs;
DROP TRIGGER IF EXISTS trigger_clarifications_updated_at ON clarifications;
DROP TRIGGER IF EXISTS trigger_work_sessions_updated_at ON work_sessions;
DROP TRIGGER IF EXISTS trigger_repositories_updated_at ON repositories;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key dependencies)
DROP TABLE IF EXISTS execution_completed_tasks;
DROP TABLE IF EXISTS task_dependencies;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS domain_events;
DROP TABLE IF EXISTS execution_validation_results;
DROP TABLE IF EXISTS task_results;
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS design_versions;
DROP TABLE IF EXISTS designs;
DROP TABLE IF EXISTS clarifications;
DROP TABLE IF EXISTS work_sessions;
DROP TABLE IF EXISTS issues;
DROP TABLE IF EXISTS repositories;

-- Drop UUID extension (optional, may want to keep it)
-- DROP EXTENSION IF EXISTS "uuid-ossp";