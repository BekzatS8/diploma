ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS reviewer_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS reviewee_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS reviewee_role TEXT,
    ADD COLUMN IF NOT EXISTS direction TEXT,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE reviews
SET reviewer_id = COALESCE(reviewer_id, client_id),
    reviewee_id = COALESCE(reviewee_id, executor_id),
    reviewee_role = COALESCE(reviewee_role, 'executor'),
    direction = COALESCE(direction, 'client_to_executor')
WHERE reviewer_id IS NULL
   OR reviewee_id IS NULL
   OR reviewee_role IS NULL
   OR direction IS NULL;

ALTER TABLE reviews
    ALTER COLUMN reviewer_id SET NOT NULL,
    ALTER COLUMN reviewee_id SET NOT NULL,
    ALTER COLUMN reviewee_role SET NOT NULL,
    ALTER COLUMN direction SET NOT NULL;

ALTER TABLE reviews
    DROP CONSTRAINT IF EXISTS reviews_reviewee_role_check,
    DROP CONSTRAINT IF EXISTS reviews_direction_check;

ALTER TABLE reviews
    ADD CONSTRAINT reviews_reviewee_role_check CHECK (reviewee_role IN ('client', 'executor')),
    ADD CONSTRAINT reviews_direction_check CHECK (direction IN ('client_to_executor', 'executor_to_client'));

ALTER TABLE reviews
    DROP CONSTRAINT IF EXISTS reviews_order_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS ux_reviews_order_reviewer_reviewee_active
    ON reviews(order_id, reviewer_id, reviewee_id)
    WHERE deleted_at IS NULL;

ALTER TABLE executor_profiles
    ALTER COLUMN rating_avg SET DEFAULT 5;

ALTER TABLE client_profiles
    ADD COLUMN IF NOT EXISTS rating_avg NUMERIC(3,2) NOT NULL DEFAULT 5 CHECK (rating_avg >= 0 AND rating_avg <= 5),
    ADD COLUMN IF NOT EXISTS rating_count INT NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    ADD COLUMN IF NOT EXISTS completed_orders INT NOT NULL DEFAULT 0 CHECK (completed_orders >= 0);

UPDATE executor_profiles
SET rating_avg = 5
WHERE rating_count = 0 AND rating_avg = 0;

INSERT INTO entity_rating_summaries(target_type, target_id, rating_avg, rating_count, updated_at)
SELECT 'user',
       reviewee_id,
       ROUND(AVG(rating)::numeric, 2),
       COUNT(*)::int,
       NOW()
FROM reviews
WHERE deleted_at IS NULL
GROUP BY reviewee_id
ON CONFLICT (target_type, target_id) DO UPDATE
SET rating_avg = EXCLUDED.rating_avg,
    rating_count = EXCLUDED.rating_count,
    updated_at = NOW();
