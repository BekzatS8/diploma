CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE entity_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    target_type TEXT NOT NULL CHECK (target_type IN (
        'user',
        'client',
        'executor',
        'coach',
        'profile',
        'order',
        'response',
        'review',
        'course',
        'course_material'
    )),
    target_id UUID NOT NULL,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_entity_reviews_target
    ON entity_reviews(target_type, target_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_entity_reviews_author
    ON entity_reviews(author_id, created_at DESC)
    WHERE deleted_at IS NULL AND author_id IS NOT NULL;

CREATE TABLE entity_rating_summaries (
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    rating_avg NUMERIC(3,2) NOT NULL DEFAULT 0 CHECK (rating_avg >= 0 AND rating_avg <= 5),
    rating_count INT NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (target_type, target_id)
);

ALTER TABLE course_materials
    ADD COLUMN IF NOT EXISTS upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX idx_course_materials_upload_id ON course_materials(upload_id)
    WHERE upload_id IS NOT NULL;

INSERT INTO users(id, email, password_hash, role, is_active, verification_status)
VALUES
    ('00000000-0000-0000-0000-000000000101', 'demo.client@buhpro.local', crypt('DemoPass123', gen_salt('bf')), 'client', TRUE, 'verified'),
    ('00000000-0000-0000-0000-000000000102', 'demo.executor@buhpro.local', crypt('DemoPass123', gen_salt('bf')), 'executor', TRUE, 'verified'),
    ('00000000-0000-0000-0000-000000000103', 'demo.coach@buhpro.local', crypt('DemoPass123', gen_salt('bf')), 'coach', TRUE, 'verified'),
    ('00000000-0000-0000-0000-000000000104', 'demo.admin@buhpro.local', crypt('DemoPass123', gen_salt('bf')), 'admin', TRUE, 'verified')
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role,
    is_active = TRUE,
    verification_status = 'verified',
    updated_at = NOW();

INSERT INTO client_profiles(user_id, company_name, tax_number, phone, about, client_type, contact_name, contact_position, address)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    'Demo Client TOO',
    '123456789012',
    '+77073271079',
    'Demo client profile for BuhPro.',
    'too',
    'Demo Client',
    'Director',
    'Astana'
)
ON CONFLICT (user_id) DO UPDATE
SET company_name = EXCLUDED.company_name,
    tax_number = EXCLUDED.tax_number,
    phone = EXCLUDED.phone,
    about = EXCLUDED.about,
    client_type = EXCLUDED.client_type,
    contact_name = EXCLUDED.contact_name,
    contact_position = EXCLUDED.contact_position,
    address = EXCLUDED.address,
    updated_at = NOW();

INSERT INTO executor_profiles(
    user_id,
    display_name,
    bio,
    years_experience,
    rating_avg,
    rating_count,
    completed_orders,
    sanction_points,
    first_name,
    last_name,
    middle_name,
    iin,
    phone,
    city,
    experience_level,
    specializations,
    education,
    work_format,
    hourly_rate,
    about,
    verification_status,
    verified_at
)
VALUES (
    '00000000-0000-0000-0000-000000000102',
    'Demo Executor',
    'Handles bookkeeping, audit and tax tasks.',
    3,
    4.50,
    2,
    1,
    0,
    'Demo',
    'Executor',
    NULL,
    '021255501034',
    '+77073271079',
    'Алматы',
    '3-5 лет',
    '["Бухгалтерский учет","Аудиторские услуги","Налоговое консультирование"]'::jsonb,
    'Higher finance education, tax certificates.',
    'remote',
    3000,
    'Demo executor profile for verification and order flows.',
    'verified',
    NOW()
)
ON CONFLICT (user_id) DO UPDATE
SET display_name = EXCLUDED.display_name,
    bio = EXCLUDED.bio,
    years_experience = EXCLUDED.years_experience,
    rating_avg = EXCLUDED.rating_avg,
    rating_count = EXCLUDED.rating_count,
    completed_orders = EXCLUDED.completed_orders,
    sanction_points = EXCLUDED.sanction_points,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    middle_name = EXCLUDED.middle_name,
    iin = EXCLUDED.iin,
    phone = EXCLUDED.phone,
    city = EXCLUDED.city,
    experience_level = EXCLUDED.experience_level,
    specializations = EXCLUDED.specializations,
    education = EXCLUDED.education,
    work_format = EXCLUDED.work_format,
    hourly_rate = EXCLUDED.hourly_rate,
    about = EXCLUDED.about,
    verification_status = 'verified',
    verified_at = COALESCE(executor_profiles.verified_at, NOW()),
    updated_at = NOW();

INSERT INTO coach_profiles(user_id, display_name, bio, expertise)
VALUES (
    '00000000-0000-0000-0000-000000000103',
    'Demo Coach',
    'Finance educator for BuhPro learning modules.',
    'Tax, bookkeeping and audit'
)
ON CONFLICT (user_id) DO UPDATE
SET display_name = EXCLUDED.display_name,
    bio = EXCLUDED.bio,
    expertise = EXCLUDED.expertise,
    updated_at = NOW();

INSERT INTO entity_reviews(author_id, target_type, target_id, rating, comment, metadata, created_at, updated_at)
SELECT client_id,
       'order',
       order_id,
       rating,
       comment,
       jsonb_build_object('legacy_review_id', id, 'executor_id', executor_id),
       created_at,
       updated_at
FROM reviews
WHERE deleted_at IS NULL;

INSERT INTO entity_reviews(author_id, target_type, target_id, rating, comment, metadata, created_at, updated_at)
SELECT client_id,
       'user',
       executor_id,
       rating,
       comment,
       jsonb_build_object('legacy_review_id', id, 'order_id', order_id),
       created_at,
       updated_at
FROM reviews
WHERE deleted_at IS NULL;

INSERT INTO entity_rating_summaries(target_type, target_id, rating_avg, rating_count, updated_at)
SELECT target_type,
       target_id,
       ROUND(AVG(rating)::numeric, 2),
       COUNT(*)::int,
       NOW()
FROM entity_reviews
WHERE deleted_at IS NULL
GROUP BY target_type, target_id
ON CONFLICT (target_type, target_id) DO UPDATE
SET rating_avg = EXCLUDED.rating_avg,
    rating_count = EXCLUDED.rating_count,
    updated_at = NOW();
