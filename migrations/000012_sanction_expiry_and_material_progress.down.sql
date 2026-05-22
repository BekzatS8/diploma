DROP INDEX IF EXISTS idx_course_material_progress_executor;
DROP INDEX IF EXISTS idx_course_material_progress_assignment;
DROP TABLE IF EXISTS course_material_progress;

DROP INDEX IF EXISTS idx_sanctions_active_ends;
ALTER TABLE sanctions DROP COLUMN IF EXISTS expired_at;
