-- 003_create_courses.sql
-- Creates the courses table linked to businesses.

CREATE TABLE IF NOT EXISTS courses (
    id          UUID PRIMARY KEY,
    business_id UUID         NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    status      VARCHAR(50)  NOT NULL DEFAULT 'draft',
    metadata    JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_courses_business_id ON courses (business_id);
