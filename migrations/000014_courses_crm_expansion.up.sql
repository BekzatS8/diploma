ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS slug TEXT,
    ADD COLUMN IF NOT EXISTS subtitle TEXT,
    ADD COLUMN IF NOT EXISTS category TEXT,
    ADD COLUMN IF NOT EXISTS level TEXT NOT NULL DEFAULT 'beginner',
    ADD COLUMN IF NOT EXISTS language TEXT NOT NULL DEFAULT 'ru',
    ADD COLUMN IF NOT EXISTS price NUMERIC(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS currency CHAR(3) NOT NULL DEFAULT 'KZT',
    ADD COLUMN IF NOT EXISTS duration_minutes INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cover_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cover_url TEXT,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS learning_outcomes TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS requirements TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS certificate_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS enrollment_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rating_avg NUMERIC(3,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rating_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS moderation_status TEXT NOT NULL DEFAULT 'approved';

ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_level_check,
    DROP CONSTRAINT IF EXISTS courses_moderation_status_check,
    DROP CONSTRAINT IF EXISTS courses_price_check,
    DROP CONSTRAINT IF EXISTS courses_duration_check;

ALTER TABLE courses
    ADD CONSTRAINT courses_level_check CHECK (level IN ('beginner', 'intermediate', 'advanced')),
    ADD CONSTRAINT courses_moderation_status_check CHECK (moderation_status IN ('draft', 'in_review', 'approved', 'rejected')),
    ADD CONSTRAINT courses_price_check CHECK (price >= 0),
    ADD CONSTRAINT courses_duration_check CHECK (duration_minutes >= 0);

UPDATE courses
SET published_at = COALESCE(published_at, updated_at)
WHERE status = 'published' AND published_at IS NULL;

UPDATE courses
SET archived_at = COALESCE(archived_at, updated_at)
WHERE status = 'archived' AND archived_at IS NULL;

ALTER TABLE course_materials
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS duration_seconds INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS is_preview BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE course_materials
    DROP CONSTRAINT IF EXISTS course_materials_duration_check;

ALTER TABLE course_materials
    ADD CONSTRAINT course_materials_duration_check CHECK (duration_seconds >= 0);

CREATE INDEX IF NOT EXISTS idx_courses_creator_status
    ON courses (created_by, status, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_courses_category_status
    ON courses (category, status, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_courses_tags
    ON courses USING GIN (tags);

CREATE UNIQUE INDEX IF NOT EXISTS ux_courses_slug_active
    ON courses (slug)
    WHERE slug IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_course_materials_preview
    ON course_materials (course_id, is_preview, sort_order);
