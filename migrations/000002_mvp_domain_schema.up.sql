CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'audit_logs' AND column_name = 'payload'
    ) THEN
        ALTER TABLE audit_logs RENAME COLUMN payload TO metadata;
    END IF;
END $$;

CREATE TYPE order_status AS ENUM (
    'draft',
    'payment_pending',
    'published',
    'in_progress',
    'completed',
    'cancelled',
    'archived'
);

CREATE TYPE response_status AS ENUM (
    'pending_payment',
    'submitted',
    'withdrawn',
    'rejected',
    'accepted'
);

CREATE TYPE payment_status AS ENUM (
    'pending',
    'succeeded',
    'failed',
    'refunded',
    'cancelled'
);

CREATE TYPE payment_object_type AS ENUM (
    'order_posting',
    'response_fee',
    'course_access',
    'promotion',
    'other'
);

CREATE TYPE sanction_status AS ENUM (
    'active',
    'resolved',
    'expired'
);

CREATE TYPE notification_status AS ENUM (
    'pending',
    'sent',
    'failed',
    'read'
);

CREATE TYPE notification_channel AS ENUM (
    'in_app',
    'email',
    'sms'
);

CREATE TYPE course_assignment_status AS ENUM (
    'assigned',
    'in_progress',
    'completed',
    'overdue',
    'cancelled'
);

CREATE TYPE message_sender_type AS ENUM (
    'user',
    'system'
);

CREATE TABLE client_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    company_name TEXT,
    tax_number TEXT,
    phone TEXT,
    about TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE executor_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT,
    bio TEXT,
    years_experience INT NOT NULL DEFAULT 0 CHECK (years_experience >= 0),
    rating_avg NUMERIC(3,2) NOT NULL DEFAULT 0 CHECK (rating_avg >= 0 AND rating_avg <= 5),
    rating_count INT NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    completed_orders INT NOT NULL DEFAULT 0 CHECK (completed_orders >= 0),
    sanction_points INT NOT NULL DEFAULT 0 CHECK (sanction_points >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE coach_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT,
    bio TEXT,
    expertise TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    category_id BIGINT REFERENCES categories(id) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    budget_amount NUMERIC(12,2) NOT NULL CHECK (budget_amount > 0),
    currency CHAR(3) NOT NULL DEFAULT 'KZT',
    status order_status NOT NULL DEFAULT 'draft',
    selected_executor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    selected_response_id UUID,
    published_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE order_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    promotion_type TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE order_status_history (
    id BIGSERIAL PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    old_status order_status,
    new_status order_status NOT NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    cover_letter TEXT,
    proposed_amount NUMERIC(12,2) CHECK (proposed_amount > 0),
    currency CHAR(3) NOT NULL DEFAULT 'KZT',
    status response_status NOT NULL DEFAULT 'pending_payment',
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (order_id, executor_id)
);

CREATE TABLE response_status_history (
    id BIGSERIAL PRIMARY KEY,
    response_id UUID NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
    old_status response_status,
    new_status response_status NOT NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_selected_response
    FOREIGN KEY (selected_response_id)
    REFERENCES responses(id)
    ON DELETE SET NULL;

CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE RESTRICT,
    client_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE sanctions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    review_id UUID REFERENCES reviews(id) ON DELETE SET NULL,
    status sanction_status NOT NULL DEFAULT 'active',
    reason TEXT NOT NULL,
    severity SMALLINT NOT NULL CHECK (severity BETWEEN 1 AND 5),
    issued_by UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE executor_rating_snapshots (
    id BIGSERIAL PRIMARY KEY,
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    review_id UUID REFERENCES reviews(id) ON DELETE SET NULL,
    rating_avg NUMERIC(3,2) NOT NULL CHECK (rating_avg >= 0 AND rating_avg <= 5),
    rating_count INT NOT NULL CHECK (rating_count >= 0),
    snapshot_reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id UUID REFERENCES users(id) ON DELETE SET NULL,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE course_materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    material_type TEXT NOT NULL CHECK (material_type IN ('video', 'article', 'file', 'link')),
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    sanction_id UUID REFERENCES sanctions(id) ON DELETE SET NULL,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT,
    status course_assignment_status NOT NULL DEFAULT 'assigned',
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL UNIQUE REFERENCES course_assignments(id) ON DELETE CASCADE,
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    progress_percent INT NOT NULL DEFAULT 0 CHECK (progress_percent BETWEEN 0 AND 100),
    status course_assignment_status NOT NULL DEFAULT 'assigned',
    last_activity_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_participants (
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_read_at TIMESTAMPTZ,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (chat_id, user_id)
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    sender_type message_sender_type NOT NULL DEFAULT 'user',
    body TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    channel notification_channel NOT NULL DEFAULT 'in_app',
    status notification_status NOT NULL DEFAULT 'pending',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    error_message TEXT
);

CREATE TABLE payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    object_type payment_object_type NOT NULL,
    object_id UUID,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    response_id UUID REFERENCES responses(id) ON DELETE SET NULL,
    provider TEXT NOT NULL,
    provider_transaction_id TEXT,
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL DEFAULT 'KZT',
    status payment_status NOT NULL DEFAULT 'pending',
    initiated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    CHECK (
        (object_type = 'order_posting' AND order_id IS NOT NULL AND response_id IS NULL)
        OR (object_type = 'response_fee' AND response_id IS NOT NULL AND order_id IS NULL)
        OR (object_type IN ('course_access', 'promotion', 'other'))
    )
);

CREATE INDEX idx_orders_active_feed ON orders (status, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_client_history ON orders (client_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_selected_executor ON orders (selected_executor_id)
    WHERE selected_executor_id IS NOT NULL;

CREATE INDEX idx_order_promotions_order ON order_promotions (order_id, starts_at DESC);
CREATE INDEX idx_order_status_history_order ON order_status_history (order_id, created_at DESC);

CREATE INDEX idx_responses_order ON responses (order_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_responses_executor ON responses (executor_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX ux_responses_one_accepted_per_order ON responses (order_id)
    WHERE status = 'accepted' AND deleted_at IS NULL;
CREATE INDEX idx_response_status_history_response ON response_status_history (response_id, created_at DESC);

CREATE INDEX idx_reviews_executor_created ON reviews (executor_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_sanctions_executor_active ON sanctions (executor_id, started_at DESC)
    WHERE status = 'active';
CREATE INDEX idx_executor_rating_snapshots_executor ON executor_rating_snapshots (executor_id, created_at DESC);

CREATE INDEX idx_courses_coach ON courses (coach_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_course_materials_course_sort ON course_materials (course_id, sort_order);
CREATE INDEX idx_course_assignments_executor_status ON course_assignments (executor_id, status, assigned_at DESC);
CREATE UNIQUE INDEX ux_course_assignments_active_per_course_executor ON course_assignments (course_id, executor_id)
    WHERE status IN ('assigned', 'in_progress');
CREATE INDEX idx_course_progress_executor_status ON course_progress (executor_id, status, updated_at DESC);

CREATE INDEX idx_messages_chat_created ON messages (chat_id, created_at);
CREATE INDEX idx_notifications_user_status ON notifications (user_id, status, created_at DESC);

CREATE UNIQUE INDEX ux_payment_provider_txn ON payment_transactions (provider, provider_transaction_id)
    WHERE provider_transaction_id IS NOT NULL;
CREATE INDEX idx_payment_user_initiated ON payment_transactions (user_id, initiated_at DESC);
CREATE INDEX idx_payment_object ON payment_transactions (object_type, object_id);
CREATE INDEX idx_payment_order ON payment_transactions (order_id)
    WHERE order_id IS NOT NULL;
CREATE INDEX idx_payment_response ON payment_transactions (response_id)
    WHERE response_id IS NOT NULL;
