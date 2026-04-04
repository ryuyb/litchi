-- T1.2.1: Initialize database schema for Litchi
-- Creates all tables for the automation development agent system

-- Enable UUID extension (required for gen_random_uuid())
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- 1. repositories table (repository configuration)
-- ============================================
CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,  -- e.g. "org/repo"
    enabled BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',           -- repository-level config override
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_repositories_name ON repositories(name);
CREATE INDEX idx_repositories_enabled ON repositories(enabled);

-- ============================================
-- 2. issues table (GitHub issue entity)
-- ============================================
CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number BIGINT NOT NULL,              -- GitHub issue number
    title VARCHAR(500) NOT NULL,
    body TEXT,
    repository VARCHAR(255) NOT NULL REFERENCES repositories(name),
    author VARCHAR(255) NOT NULL,        -- GitHub username of issue author
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(repository, number)
);

CREATE INDEX idx_issues_repository ON issues(repository);
CREATE INDEX idx_issues_number ON issues(number);
CREATE INDEX idx_issues_author ON issues(author);

-- ============================================
-- 3. work_sessions table (aggregate root)
-- ============================================
CREATE TABLE work_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    current_stage VARCHAR(50) NOT NULL DEFAULT 'clarification',
    -- stages: clarification, design, task_breakdown, execution, pull_request, completed
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    -- status: active, paused, terminated, completed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_work_sessions_issue_id ON work_sessions(issue_id);
CREATE INDEX idx_work_sessions_status ON work_sessions(status);
CREATE INDEX idx_work_sessions_current_stage ON work_sessions(current_stage);

-- ============================================
-- 4. clarifications table (clarification entity)
-- ============================================
CREATE TABLE clarifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    confirmed_points JSONB DEFAULT '[]',     -- list of confirmed requirement points
    pending_questions JSONB DEFAULT '[]',    -- list of pending questions to answer
    conversation_history JSONB DEFAULT '[]', -- conversation turns history
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress',
    -- status: in_progress, completed
    clarity_score INT,                       -- overall clarity score (0-100)
    clarity_dimensions JSONB,                -- detailed clarity dimension scores
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_clarifications_session_id ON clarifications(session_id);
CREATE INDEX idx_clarifications_status ON clarifications(status);

-- ============================================
-- 5. designs table (design entity)
-- ============================================
CREATE TABLE designs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    current_version INT NOT NULL DEFAULT 0,
    complexity_score INT,                    -- complexity score (0-100)
    require_confirmation BOOLEAN DEFAULT false,
    confirmed BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_designs_session_id ON designs(session_id);

-- ============================================
-- 6. design_versions table (design version history)
-- ============================================
CREATE TABLE design_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    design_id UUID NOT NULL REFERENCES designs(id) ON DELETE CASCADE,
    version INT NOT NULL,
    content TEXT NOT NULL,                   -- design document content
    reason VARCHAR(500),                     -- reason for version change
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(design_id, version)
);

CREATE INDEX idx_design_versions_design_id ON design_versions(design_id);

-- ============================================
-- 7. tasks table (task entity)
-- ============================================
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- status: pending, running, completed, failed, skipped, retrying
    retry_count INT DEFAULT 0,
    failure_reason TEXT,
    suggestion TEXT,                         -- suggested fix for failed task
    seq INT NOT NULL,                        -- execution sequence
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tasks_session_id ON tasks(session_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_seq ON tasks(session_id, seq);

-- ============================================
-- 7.1 task_dependencies table (task dependency relation)
-- ============================================
CREATE TABLE task_dependencies (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, depends_on_task_id)
);

CREATE INDEX idx_task_dependencies_task ON task_dependencies(task_id);
CREATE INDEX idx_task_dependencies_depends_on ON task_dependencies(depends_on_task_id);

-- ============================================
-- 8. executions table (execution entity)
-- ============================================
CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    worktree_path VARCHAR(500),              -- git worktree path
    branch_name VARCHAR(255),                -- current branch name
    branch_deprecated BOOLEAN DEFAULT false, -- whether branch is deprecated
    deprecated_branches JSONB DEFAULT '[]',  -- history of deprecated branches
    current_task_id UUID,                    -- currently executing task
    failed_task JSONB,                       -- failed task details: {taskId, reason, suggestion}
    fix_tasks JSONB DEFAULT '[]',            -- fix tasks added on PR rollback
    rollback_history JSONB DEFAULT '[]',     -- rollback operation history
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_executions_session_id ON executions(session_id);

-- ============================================
-- 8.1 execution_completed_tasks table (completed tasks relation)
-- ============================================
CREATE TABLE execution_completed_tasks (
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (execution_id, task_id)
);

CREATE INDEX idx_execution_completed_tasks_execution ON execution_completed_tasks(execution_id);
CREATE INDEX idx_execution_completed_tasks_task ON execution_completed_tasks(task_id);

-- ============================================
-- 9. task_results table (task execution result)
-- ============================================
CREATE TABLE task_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    output TEXT,                             -- execution output
    test_results JSONB DEFAULT '[]',         -- test result details
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_task_results_task_id ON task_results(task_id);

-- ============================================
-- 10. execution_validation_results table
-- ============================================
CREATE TABLE execution_validation_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,

    -- Formatting results
    format_success BOOLEAN,
    format_output TEXT,
    format_duration_ms BIGINT,

    -- Lint results
    lint_success BOOLEAN,
    lint_output TEXT,
    lint_issues_found INT,
    lint_issues_fixed INT,
    lint_duration_ms BIGINT,

    -- Test results
    test_success BOOLEAN,
    test_output TEXT,
    test_passed INT,
    test_failed INT,
    test_duration_ms BIGINT,

    -- Overall result
    overall_status VARCHAR(50) NOT NULL,  -- passed / failed / warned / skipped
    warnings JSONB DEFAULT '[]',

    -- Timing
    total_duration_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_validation_results_session ON execution_validation_results(session_id);
CREATE INDEX idx_validation_results_task ON execution_validation_results(task_id);
CREATE INDEX idx_validation_results_status ON execution_validation_results(overall_status);

-- ============================================
-- 11. domain_events table (event store)
-- ============================================
CREATE TABLE domain_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,    -- e.g. "WorkSession"
    event_type VARCHAR(100) NOT NULL,        -- e.g. "WorkSessionStarted"
    payload JSONB NOT NULL,
    occurred_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_domain_events_aggregate ON domain_events(aggregate_id, aggregate_type);
CREATE INDEX idx_domain_events_type ON domain_events(event_type);
CREATE INDEX idx_domain_events_occurred_at ON domain_events(occurred_at);

-- ============================================
-- 12. audit_logs table (audit trail)
-- ============================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    session_id UUID REFERENCES work_sessions(id) ON DELETE SET NULL,
    repository VARCHAR(255) NOT NULL,
    issue_number BIGINT,

    -- Actor information
    actor VARCHAR(255) NOT NULL,             -- GitHub username
    actor_role VARCHAR(50),                  -- admin / issue_author

    -- Operation details
    operation VARCHAR(100) NOT NULL,         -- operation type
    resource_type VARCHAR(100),              -- resource type
    resource_id VARCHAR(255),                -- resource identifier

    -- Result
    result VARCHAR(50) NOT NULL,             -- success / failed / denied
    duration_ms BIGINT,                      -- operation duration in milliseconds

    -- Details
    parameters JSONB,                        -- operation parameters
    output TEXT,                             -- output summary (truncated)
    error_message TEXT,                      -- error message

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_session ON audit_logs(session_id);
CREATE INDEX idx_audit_logs_repository ON audit_logs(repository);
CREATE INDEX idx_audit_logs_operation ON audit_logs(operation);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor);
CREATE INDEX idx_audit_logs_result ON audit_logs(result);

-- ============================================
-- 13. webhook_deliveries table (idempotency)
-- ============================================
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id VARCHAR(255) NOT NULL UNIQUE, -- GitHub delivery ID (X-GitHub-Delivery)
    event_type VARCHAR(100) NOT NULL,          -- e.g. issues, issue_comment
    repository VARCHAR(255) NOT NULL,          -- repository name
    payload_hash VARCHAR(64),                  -- payload SHA256 hash
    processed BOOLEAN DEFAULT false,           -- whether processed
    process_result VARCHAR(50),                -- success / ignored / error
    process_message TEXT,                      -- processing message
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE        -- expiration time (default 24h)
);

CREATE INDEX idx_webhook_deliveries_delivery_id ON webhook_deliveries(delivery_id);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);
CREATE INDEX idx_webhook_deliveries_expires_at ON webhook_deliveries(expires_at);
CREATE INDEX idx_webhook_deliveries_processed ON webhook_deliveries(processed, expires_at);

-- ============================================
-- Updated_at trigger function (for auto-update)
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to tables with updated_at
CREATE TRIGGER trigger_repositories_updated_at
    BEFORE UPDATE ON repositories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_work_sessions_updated_at
    BEFORE UPDATE ON work_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_clarifications_updated_at
    BEFORE UPDATE ON clarifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_designs_updated_at
    BEFORE UPDATE ON designs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_executions_updated_at
    BEFORE UPDATE ON executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();