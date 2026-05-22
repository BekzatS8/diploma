ALTER TABLE sanctions
    ADD COLUMN IF NOT EXISTS expired_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_sanctions_active_ends
    ON sanctions (ends_at)
    WHERE status = 'active' AND ends_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS course_material_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES course_assignments(id) ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES course_materials(id) ON DELETE CASCADE,
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assignment_id, material_id)
);

CREATE INDEX IF NOT EXISTS idx_course_material_progress_assignment
    ON course_material_progress (assignment_id, completed_at DESC);

CREATE INDEX IF NOT EXISTS idx_course_material_progress_executor
    ON course_material_progress (executor_id, completed_at DESC);
