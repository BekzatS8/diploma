UPDATE payment_transactions
SET object_type = 'response_fee'
WHERE object_type = 'response_submission';

UPDATE responses SET status = 'withdrawn'::response_status WHERE status = 'cancelled';
UPDATE responses SET status = 'pending_payment'::response_status WHERE status IN ('draft', 'payment_pending');

UPDATE response_status_history SET old_status = 'pending_payment'::response_status WHERE old_status = 'payment_pending';
UPDATE response_status_history SET new_status = 'pending_payment'::response_status WHERE new_status = 'payment_pending';
UPDATE response_status_history SET old_status = 'withdrawn'::response_status WHERE old_status = 'cancelled';
UPDATE response_status_history SET new_status = 'withdrawn'::response_status WHERE new_status = 'cancelled';

ALTER TABLE payment_transactions DROP CONSTRAINT IF EXISTS payment_transactions_check;
ALTER TABLE payment_transactions
    ADD CONSTRAINT payment_transactions_check
    CHECK (
        (object_type = 'order_posting' AND order_id IS NOT NULL AND response_id IS NULL)
        OR (object_type = 'response_fee' AND response_id IS NOT NULL AND order_id IS NULL)
        OR (object_type IN ('course_access', 'promotion', 'other'))
    );
