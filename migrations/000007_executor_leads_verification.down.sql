DROP INDEX IF EXISTS idx_executor_lead_documents_lead;
DROP TABLE IF EXISTS executor_lead_documents;

DROP INDEX IF EXISTS idx_executor_leads_converted_user;
DROP INDEX IF EXISTS idx_executor_leads_status_created;
DROP INDEX IF EXISTS ux_executor_leads_email_open;
DROP TABLE IF EXISTS executor_leads;

DROP INDEX IF EXISTS idx_executor_profiles_specializations;
DROP INDEX IF EXISTS idx_executor_profiles_city;
DROP INDEX IF EXISTS idx_executor_profiles_verification_status;

ALTER TABLE executor_profiles DROP CONSTRAINT IF EXISTS executor_profiles_iin_check;
ALTER TABLE executor_profiles DROP CONSTRAINT IF EXISTS executor_profiles_verification_status_check;
ALTER TABLE executor_profiles
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS rejected_at,
    DROP COLUMN IF EXISTS verified_at,
    DROP COLUMN IF EXISTS verification_status,
    DROP COLUMN IF EXISTS about,
    DROP COLUMN IF EXISTS hourly_rate,
    DROP COLUMN IF EXISTS work_format,
    DROP COLUMN IF EXISTS education,
    DROP COLUMN IF EXISTS specializations,
    DROP COLUMN IF EXISTS experience_level,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS iin,
    DROP COLUMN IF EXISTS middle_name,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS first_name;

ALTER TABLE client_profiles DROP CONSTRAINT IF EXISTS client_profiles_client_type_check;
ALTER TABLE client_profiles
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS contact_position,
    DROP COLUMN IF EXISTS contact_name,
    DROP COLUMN IF EXISTS client_type;

DROP INDEX IF EXISTS idx_users_verification_status;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_verification_status_check;
ALTER TABLE users
    DROP COLUMN IF EXISTS verification_rejection_reason,
    DROP COLUMN IF EXISTS verification_reviewed_by,
    DROP COLUMN IF EXISTS verification_reviewed_at,
    DROP COLUMN IF EXISTS verification_submitted_at,
    DROP COLUMN IF EXISTS verification_status;
