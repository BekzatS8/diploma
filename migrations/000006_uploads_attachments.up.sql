CREATE TABLE uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_uploads_author_created ON uploads(author_id, created_at DESC);

CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN (
        'profile_document',
        'order_attachment',
        'response_attachment',
        'review_attachment',
        'chat_attachment',
        'course_material'
    )),
    target_id UUID NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (upload_id, target_type, target_id)
);

CREATE INDEX idx_attachments_target ON attachments(target_type, target_id, sort_order, created_at);
CREATE INDEX idx_attachments_upload ON attachments(upload_id);

ALTER TABLE client_profiles
    ADD COLUMN avatar_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL;

ALTER TABLE executor_profiles
    ADD COLUMN avatar_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL;

ALTER TABLE coach_profiles
    ADD COLUMN avatar_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX idx_client_profiles_avatar_upload_id ON client_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;
CREATE INDEX idx_executor_profiles_avatar_upload_id ON executor_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;
CREATE INDEX idx_coach_profiles_avatar_upload_id ON coach_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;
