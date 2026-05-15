DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS wallets;

DROP INDEX IF EXISTS idx_orders_region;
DROP INDEX IF EXISTS idx_orders_deadline_at;
DROP INDEX IF EXISTS idx_orders_public_promoted;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_status_check;
ALTER TABLE orders
    DROP COLUMN IF EXISTS executor_paid_at,
    DROP COLUMN IF EXISTS highlighted_until,
    DROP COLUMN IF EXISTS pinned_until,
    DROP COLUMN IF EXISTS promoted_until,
    DROP COLUMN IF EXISTS payment_status,
    DROP COLUMN IF EXISTS total_charge,
    DROP COLUMN IF EXISTS escrow_amount,
    DROP COLUMN IF EXISTS promotion_fee,
    DROP COLUMN IF EXISTS posting_fee,
    DROP COLUMN IF EXISTS promotion_options,
    DROP COLUMN IF EXISTS region,
    DROP COLUMN IF EXISTS deadline_at;

ALTER TABLE coach_profiles DROP COLUMN IF EXISTS website;

ALTER TABLE executor_profiles
    DROP COLUMN IF EXISTS response_rate,
    DROP COLUMN IF EXISTS profile_views,
    DROP COLUMN IF EXISTS website;

ALTER TABLE client_profiles DROP COLUMN IF EXISTS website;
