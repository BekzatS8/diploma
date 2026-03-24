ALTER TABLE course_assignments DROP CONSTRAINT IF EXISTS course_assignments_source_check;
ALTER TABLE course_assignments DROP COLUMN IF EXISTS source;

ALTER TABLE course_materials DROP CONSTRAINT IF EXISTS course_materials_material_type_check;
ALTER TABLE course_materials ADD CONSTRAINT course_materials_material_type_check CHECK (material_type IN ('video', 'article', 'file', 'link'));
ALTER TABLE course_materials DROP COLUMN IF EXISTS updated_at;
ALTER TABLE course_materials DROP COLUMN IF EXISTS content;
ALTER TABLE course_materials ALTER COLUMN url SET NOT NULL;

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_status_check;
ALTER TABLE courses DROP COLUMN IF EXISTS status;
ALTER TABLE courses DROP COLUMN IF EXISTS created_by;
