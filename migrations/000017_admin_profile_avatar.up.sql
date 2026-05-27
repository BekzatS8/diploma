ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_avatar_upload_id ON users(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;
