ALTER TABLE users
    ADD COLUMN IF NOT EXISTS verification_status TEXT NOT NULL DEFAULT 'verified',
    ADD COLUMN IF NOT EXISTS verification_submitted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verification_reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verification_reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS verification_rejection_reason TEXT;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_verification_status_check;
ALTER TABLE users
    ADD CONSTRAINT users_verification_status_check
    CHECK (verification_status IN ('none', 'pending', 'in_review', 'verified', 'rejected'));

CREATE INDEX IF NOT EXISTS idx_users_verification_status ON users(verification_status);

ALTER TABLE client_profiles
    ADD COLUMN IF NOT EXISTS client_type TEXT,
    ADD COLUMN IF NOT EXISTS contact_name TEXT,
    ADD COLUMN IF NOT EXISTS contact_position TEXT,
    ADD COLUMN IF NOT EXISTS address TEXT;

ALTER TABLE client_profiles DROP CONSTRAINT IF EXISTS client_profiles_client_type_check;
ALTER TABLE client_profiles
    ADD CONSTRAINT client_profiles_client_type_check
    CHECK (client_type IS NULL OR client_type IN ('too', 'ip', 'representative'));

ALTER TABLE executor_profiles
    ADD COLUMN IF NOT EXISTS first_name TEXT,
    ADD COLUMN IF NOT EXISTS last_name TEXT,
    ADD COLUMN IF NOT EXISTS middle_name TEXT,
    ADD COLUMN IF NOT EXISTS iin TEXT,
    ADD COLUMN IF NOT EXISTS phone TEXT,
    ADD COLUMN IF NOT EXISTS city TEXT,
    ADD COLUMN IF NOT EXISTS experience_level TEXT,
    ADD COLUMN IF NOT EXISTS specializations JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS education TEXT,
    ADD COLUMN IF NOT EXISTS work_format TEXT,
    ADD COLUMN IF NOT EXISTS hourly_rate NUMERIC(12,2),
    ADD COLUMN IF NOT EXISTS about TEXT,
    ADD COLUMN IF NOT EXISTS verification_status TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejection_reason TEXT;

ALTER TABLE executor_profiles DROP CONSTRAINT IF EXISTS executor_profiles_verification_status_check;
ALTER TABLE executor_profiles
    ADD CONSTRAINT executor_profiles_verification_status_check
    CHECK (verification_status IN ('none', 'pending', 'in_review', 'verified', 'rejected'));

ALTER TABLE executor_profiles DROP CONSTRAINT IF EXISTS executor_profiles_iin_check;
ALTER TABLE executor_profiles
    ADD CONSTRAINT executor_profiles_iin_check
    CHECK (iin IS NULL OR length(iin) = 12);

CREATE INDEX IF NOT EXISTS idx_executor_profiles_verification_status ON executor_profiles(verification_status);
CREATE INDEX IF NOT EXISTS idx_executor_profiles_city ON executor_profiles(city) WHERE city IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_profiles_specializations ON executor_profiles USING GIN(specializations);

CREATE TABLE executor_leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    middle_name TEXT,
    iin TEXT NOT NULL CHECK (length(iin) = 12),
    phone TEXT NOT NULL,
    city TEXT NOT NULL,
    experience_level TEXT NOT NULL,
    specializations JSONB NOT NULL DEFAULT '[]'::jsonb,
    education TEXT NOT NULL,
    work_format TEXT,
    hourly_rate NUMERIC(12,2) CHECK (hourly_rate IS NULL OR hourly_rate >= 0),
    about TEXT NOT NULL,
    terms_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'in_review', 'approved', 'rejected', 'converted')),
    priority INT NOT NULL DEFAULT 0 CHECK (priority BETWEEN 0 AND 2),
    notes TEXT,
    rejection_reason TEXT,
    source TEXT,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    ip_address TEXT,
    user_agent TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    converted_at TIMESTAMPTZ,
    converted_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_executor_leads_email_open
    ON executor_leads (LOWER(email))
    WHERE status IN ('new', 'in_review', 'approved');
CREATE INDEX idx_executor_leads_status_created ON executor_leads(status, created_at DESC);
CREATE INDEX idx_executor_leads_converted_user ON executor_leads(converted_user_id)
    WHERE converted_user_id IS NOT NULL;

CREATE TABLE executor_lead_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL REFERENCES executor_leads(id) ON DELETE CASCADE,
    document_type TEXT NOT NULL CHECK (document_type IN ('identity', 'education', 'ip_registration', 'other')),
    file_path TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_executor_lead_documents_lead ON executor_lead_documents(lead_id, document_type, created_at);
