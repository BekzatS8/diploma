DROP INDEX IF EXISTS idx_users_avatar_upload_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS avatar_upload_id;
