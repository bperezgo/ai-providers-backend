-- 004_create_course_chapters.sql
-- Creates the course_chapters table for chapter content and production plans.

CREATE TABLE IF NOT EXISTS course_chapters (
    id         UUID PRIMARY KEY,
    course_id  UUID         NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    number     INTEGER      NOT NULL,
    title      VARCHAR(255) NOT NULL,
    content    JSONB,
    plan       JSONB,
    status     VARCHAR(50)  NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_course_chapters_course_id ON course_chapters (course_id);
