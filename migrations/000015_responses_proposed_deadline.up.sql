ALTER TABLE responses
    ADD COLUMN IF NOT EXISTS proposed_deadline TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_responses_proposed_deadline
    ON responses(proposed_deadline)
    WHERE proposed_deadline IS NOT NULL AND deleted_at IS NULL;
