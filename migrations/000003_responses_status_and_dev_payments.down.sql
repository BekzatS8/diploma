-- Enum values cannot be removed safely with ALTER TYPE ... DROP VALUE in PostgreSQL.
-- Data rollback is handled in 000004_responses_status_and_dev_payments_data_and_checks.down.sql
-- before this migration is reverted.
SELECT 1;
