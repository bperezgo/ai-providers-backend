-- 001_create_jobs.sql
-- Creates the jobs table for AI generation job tracking.

CREATE TABLE IF NOT EXISTS jobs (
    id              UUID PRIMARY KEY,
    provider        VARCHAR(50)  NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    progress        INTEGER      NOT NULL DEFAULT 0,
    provider_job_id VARCHAR(255),
    request_data    JSONB,
    result_url      TEXT,
    result_data     JSONB,
    error_message   TEXT,
    entity_type     VARCHAR(50),
    entity_id       UUID,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_provider   ON jobs (provider);
CREATE INDEX IF NOT EXISTS idx_jobs_status     ON jobs (status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs (created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_expires_at ON jobs (expires_at);
CREATE INDEX IF NOT EXISTS idx_jobs_entity     ON jobs (entity_type, entity_id);
