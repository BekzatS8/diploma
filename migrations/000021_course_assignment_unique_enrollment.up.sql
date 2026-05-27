-- Remove duplicate non-cancelled assignments, keep the newest per course+executor
DELETE FROM course_assignments ca
USING course_assignments newer
WHERE ca.course_id = newer.course_id
  AND ca.executor_id = newer.executor_id
  AND ca.status <> 'cancelled'
  AND newer.status <> 'cancelled'
  AND ca.created_at < newer.created_at;

DROP INDEX IF EXISTS ux_course_assignments_active_per_course_executor;

CREATE UNIQUE INDEX ux_course_assignments_per_course_executor
    ON course_assignments (course_id, executor_id)
    WHERE status <> 'cancelled';
