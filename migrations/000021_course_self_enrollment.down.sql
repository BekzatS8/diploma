DROP INDEX IF EXISTS idx_course_assignments_course_executor_status;

ALTER TABLE course_assignments
    DROP CONSTRAINT IF EXISTS course_assignments_source_check;

ALTER TABLE course_assignments
    ADD CONSTRAINT course_assignments_source_check
    CHECK (source IN (
        'manual_admin',
        'sanction_low_rating_first',
        'sanction_low_rating_repeat'
    ));
