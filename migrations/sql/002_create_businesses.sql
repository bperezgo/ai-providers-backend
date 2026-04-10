-- 002_create_businesses.sql
-- Creates the businesses table for multi-business management.

CREATE TABLE IF NOT EXISTS businesses (
    id          UUID PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    type        VARCHAR(100),
    location    VARCHAR(500),
    description TEXT,
    discovery   JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_businesses_slug ON businesses (slug);
