ALTER TABLE client_profiles
    ADD COLUMN IF NOT EXISTS website TEXT;

ALTER TABLE executor_profiles
    ADD COLUMN IF NOT EXISTS website TEXT,
    ADD COLUMN IF NOT EXISTS profile_views INT NOT NULL DEFAULT 0 CHECK (profile_views >= 0),
    ADD COLUMN IF NOT EXISTS response_rate INT NOT NULL DEFAULT 0 CHECK (response_rate BETWEEN 0 AND 100);

ALTER TABLE coach_profiles
    ADD COLUMN IF NOT EXISTS website TEXT;

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS deadline_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS region TEXT,
    ADD COLUMN IF NOT EXISTS promotion_options JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS posting_fee NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (posting_fee >= 0),
    ADD COLUMN IF NOT EXISTS promotion_fee NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (promotion_fee >= 0),
    ADD COLUMN IF NOT EXISTS escrow_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (escrow_amount >= 0),
    ADD COLUMN IF NOT EXISTS total_charge NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (total_charge >= 0),
    ADD COLUMN IF NOT EXISTS payment_status TEXT NOT NULL DEFAULT 'unpaid',
    ADD COLUMN IF NOT EXISTS promoted_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS pinned_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS highlighted_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS executor_paid_at TIMESTAMPTZ;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_status_check;
ALTER TABLE orders
    ADD CONSTRAINT orders_payment_status_check
    CHECK (payment_status IN ('unpaid', 'paid', 'refunded', 'released'));

CREATE INDEX IF NOT EXISTS idx_orders_public_promoted
    ON orders(status, pinned_until DESC, promoted_until DESC, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_orders_deadline_at ON orders(deadline_at)
    WHERE deadline_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_orders_region ON orders(region)
    WHERE region IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE wallets (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'KZT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    direction TEXT NOT NULL CHECK (direction IN ('credit', 'debit')),
    currency CHAR(3) NOT NULL DEFAULT 'KZT',
    reason TEXT NOT NULL,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX idx_wallet_transactions_user_created
    ON wallet_transactions(user_id, created_at DESC);
CREATE INDEX idx_wallet_transactions_order
    ON wallet_transactions(order_id)
    WHERE order_id IS NOT NULL;

INSERT INTO wallets(user_id, balance, currency)
SELECT id, 0, 'KZT'
FROM users
ON CONFLICT (user_id) DO NOTHING;

UPDATE wallets
SET balance = 250000, updated_at = NOW()
WHERE user_id = '00000000-0000-0000-0000-000000000101' AND balance = 0;

UPDATE wallets
SET balance = 50000, updated_at = NOW()
WHERE user_id = '00000000-0000-0000-0000-000000000102' AND balance = 0;

UPDATE client_profiles
SET website = COALESCE(website, 'https://client-demo.buhpro.local')
WHERE user_id = '00000000-0000-0000-0000-000000000101';

UPDATE executor_profiles
SET website = COALESCE(website, 'https://executor-demo.buhpro.local'),
    profile_views = GREATEST(profile_views, 234),
    response_rate = GREATEST(response_rate, 85)
WHERE user_id = '00000000-0000-0000-0000-000000000102';
