DROP INDEX IF EXISTS ux_reviews_order_reviewer_reviewee_active;

ALTER TABLE reviews
    DROP CONSTRAINT IF EXISTS reviews_direction_check,
    DROP CONSTRAINT IF EXISTS reviews_reviewee_role_check,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS direction,
    DROP COLUMN IF EXISTS reviewee_role,
    DROP COLUMN IF EXISTS reviewee_id,
    DROP COLUMN IF EXISTS reviewer_id;

ALTER TABLE client_profiles
    DROP COLUMN IF EXISTS completed_orders,
    DROP COLUMN IF EXISTS rating_count,
    DROP COLUMN IF EXISTS rating_avg;

ALTER TABLE executor_profiles
    ALTER COLUMN rating_avg SET DEFAULT 0;
