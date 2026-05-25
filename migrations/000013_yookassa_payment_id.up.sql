ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS yookassa_payment_id TEXT;

UPDATE payment_transactions
SET yookassa_payment_id = provider_transaction_id
WHERE provider = 'yookassa'
  AND provider_transaction_id IS NOT NULL
  AND yookassa_payment_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_payment_yookassa_payment_id
    ON payment_transactions (yookassa_payment_id)
    WHERE yookassa_payment_id IS NOT NULL;
