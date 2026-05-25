DROP INDEX IF EXISTS ux_payment_yookassa_payment_id;

ALTER TABLE payment_transactions
    DROP COLUMN IF EXISTS yookassa_payment_id;
