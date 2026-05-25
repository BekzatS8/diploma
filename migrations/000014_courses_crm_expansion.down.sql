DROP INDEX IF EXISTS idx_course_materials_preview;
DROP INDEX IF EXISTS ux_courses_slug_active;
DROP INDEX IF EXISTS idx_courses_tags;
DROP INDEX IF EXISTS idx_courses_category_status;
DROP INDEX IF EXISTS idx_courses_creator_status;

ALTER TABLE course_materials
    DROP CONSTRAINT IF EXISTS course_materials_duration_check,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS is_preview,
    DROP COLUMN IF EXISTS duration_seconds,
    DROP COLUMN IF EXISTS description;

ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_duration_check,
    DROP CONSTRAINT IF EXISTS courses_price_check,
    DROP CONSTRAINT IF EXISTS courses_moderation_status_check,
    DROP CONSTRAINT IF EXISTS courses_level_check,
    DROP COLUMN IF EXISTS moderation_status,
    DROP COLUMN IF EXISTS rating_count,
    DROP COLUMN IF EXISTS rating_avg,
    DROP COLUMN IF EXISTS enrollment_count,
    DROP COLUMN IF EXISTS archived_at,
    DROP COLUMN IF EXISTS published_at,
    DROP COLUMN IF EXISTS certificate_enabled,
    DROP COLUMN IF EXISTS requirements,
    DROP COLUMN IF EXISTS learning_outcomes,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS cover_url,
    DROP COLUMN IF EXISTS cover_upload_id,
    DROP COLUMN IF EXISTS duration_minutes,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS price,
    DROP COLUMN IF EXISTS language,
    DROP COLUMN IF EXISTS level,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS subtitle,
    DROP COLUMN IF EXISTS slug;
