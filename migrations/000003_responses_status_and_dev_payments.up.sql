DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid
        WHERE t.typname='response_status' AND e.enumlabel='draft'
    ) THEN
        ALTER TYPE response_status ADD VALUE 'draft';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid
        WHERE t.typname='response_status' AND e.enumlabel='payment_pending'
    ) THEN
        ALTER TYPE response_status ADD VALUE 'payment_pending';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid
        WHERE t.typname='response_status' AND e.enumlabel='cancelled'
    ) THEN
        ALTER TYPE response_status ADD VALUE 'cancelled';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid
        WHERE t.typname='payment_object_type' AND e.enumlabel='response_submission'
    ) THEN
        ALTER TYPE payment_object_type ADD VALUE 'response_submission';
    END IF;
END $$;

ALTER TABLE payment_transactions DROP CONSTRAINT IF EXISTS payment_transactions_check;
ALTER TABLE payment_transactions
    ADD CONSTRAINT payment_transactions_check
    CHECK (
        (object_type = 'order_posting' AND order_id IS NOT NULL AND response_id IS NULL)
        OR (object_type IN ('response_fee', 'response_submission') AND response_id IS NOT NULL AND order_id IS NULL)
        OR (object_type IN ('course_access', 'promotion', 'other'))
    );

UPDATE payment_transactions
SET object_type = 'response_submission'
WHERE object_type = 'response_fee';

UPDATE responses r
SET status = CASE
    WHEN EXISTS (
        SELECT 1
        FROM payment_transactions pt
        WHERE pt.response_id = r.id
          AND pt.object_type = 'response_submission'
          AND pt.status = 'pending'
    ) THEN 'payment_pending'::response_status
    ELSE 'draft'::response_status
END
WHERE r.status = 'pending_payment';

UPDATE responses SET status = 'cancelled'::response_status WHERE status = 'withdrawn';
UPDATE response_status_history SET new_status = 'payment_pending'::response_status WHERE new_status = 'pending_payment';
UPDATE response_status_history SET old_status = 'payment_pending'::response_status WHERE old_status = 'pending_payment';
UPDATE response_status_history SET new_status = 'cancelled'::response_status WHERE new_status = 'withdrawn';
UPDATE response_status_history SET old_status = 'cancelled'::response_status WHERE old_status = 'withdrawn';
