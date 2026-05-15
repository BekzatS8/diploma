DROP INDEX IF EXISTS idx_coach_profiles_avatar_upload_id;
DROP INDEX IF EXISTS idx_executor_profiles_avatar_upload_id;
DROP INDEX IF EXISTS idx_client_profiles_avatar_upload_id;

ALTER TABLE coach_profiles DROP COLUMN IF EXISTS avatar_upload_id;
ALTER TABLE executor_profiles DROP COLUMN IF EXISTS avatar_upload_id;
ALTER TABLE client_profiles DROP COLUMN IF EXISTS avatar_upload_id;

DROP INDEX IF EXISTS idx_attachments_upload;
DROP INDEX IF EXISTS idx_attachments_target;
DROP TABLE IF EXISTS attachments;

DROP INDEX IF EXISTS idx_uploads_author_created;
DROP TABLE IF EXISTS uploads;
