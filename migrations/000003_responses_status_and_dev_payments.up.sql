DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'response_status' AND e.enumlabel = 'draft'
    ) THEN
        ALTER TYPE response_status ADD VALUE 'draft';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'response_status' AND e.enumlabel = 'payment_pending'
    ) THEN
        ALTER TYPE response_status ADD VALUE 'payment_pending';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'response_status' AND e.enumlabel = 'cancelled'
    ) THEN
        ALTER TYPE response_status ADD VALUE 'cancelled';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'payment_object_type' AND e.enumlabel = 'response_submission'
    ) THEN
        ALTER TYPE payment_object_type ADD VALUE 'response_submission';
    END IF;
END $$;
