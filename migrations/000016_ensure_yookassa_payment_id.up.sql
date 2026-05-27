ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS yookassa_payment_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS ux_payment_yookassa_payment_id
    ON payment_transactions (yookassa_payment_id)
    WHERE yookassa_payment_id IS NOT NULL;
