ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS status TEXT;

UPDATE courses
SET created_by = COALESCE(created_by, coach_id)
WHERE created_by IS NULL;

UPDATE courses
SET status = CASE
    WHEN deleted_at IS NOT NULL THEN 'archived'
    WHEN is_published THEN 'published'
    ELSE 'draft'
END
WHERE status IS NULL;

ALTER TABLE courses
    ALTER COLUMN status SET DEFAULT 'draft',
    ALTER COLUMN status SET NOT NULL;

ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_status_check;

ALTER TABLE courses
    ADD CONSTRAINT courses_status_check CHECK (status IN ('draft', 'published', 'archived'));

ALTER TABLE course_materials
    ALTER COLUMN url DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS content TEXT,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE course_materials
    DROP CONSTRAINT IF EXISTS course_materials_material_type_check;

ALTER TABLE course_materials
    ADD CONSTRAINT course_materials_material_type_check CHECK (material_type IN ('video', 'pdf', 'link', 'text'));

ALTER TABLE course_assignments
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'manual_admin';

ALTER TABLE course_assignments
    DROP CONSTRAINT IF EXISTS course_assignments_source_check;

ALTER TABLE course_assignments
    ADD CONSTRAINT course_assignments_source_check CHECK (source IN ('manual_admin', 'sanction_low_rating_first', 'sanction_low_rating_repeat'));
