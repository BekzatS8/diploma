ALTER TABLE course_assignments
    DROP CONSTRAINT IF EXISTS course_assignments_source_check;

UPDATE course_assignments
SET source = 'self_enrolled'
WHERE source = 'self_enroll';

ALTER TABLE course_assignments
    ADD CONSTRAINT course_assignments_source_check
    CHECK (source IN (
        'manual_admin',
        'self_enrolled',
        'sanction_low_rating_first',
        'sanction_low_rating_repeat'
    ));

CREATE INDEX IF NOT EXISTS idx_course_assignments_course_executor_status
    ON course_assignments(course_id, executor_id, status, assigned_at DESC);

UPDATE courses c
SET enrollment_count = COALESCE(stats.enrollment_count, 0),
    updated_at = NOW()
FROM (
    SELECT course_id, COUNT(*)::int AS enrollment_count
    FROM course_assignments
    WHERE status <> 'cancelled'
    GROUP BY course_id
) stats
WHERE c.id = stats.course_id;

UPDATE courses c
SET enrollment_count = 0,
    updated_at = NOW()
WHERE NOT EXISTS (
    SELECT 1
    FROM course_assignments ca
    WHERE ca.course_id = c.id
      AND ca.status <> 'cancelled'
);
