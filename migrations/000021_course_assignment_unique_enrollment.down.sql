DROP INDEX IF EXISTS ux_course_assignments_per_course_executor;

CREATE UNIQUE INDEX ux_course_assignments_active_per_course_executor
    ON course_assignments (course_id, executor_id)
    WHERE status IN ('assigned', 'in_progress');
