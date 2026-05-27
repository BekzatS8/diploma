DROP INDEX IF EXISTS idx_responses_proposed_deadline;

ALTER TABLE responses
    DROP COLUMN IF EXISTS proposed_deadline;
