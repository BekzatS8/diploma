DROP INDEX IF EXISTS idx_payment_response;
DROP INDEX IF EXISTS idx_payment_order;
DROP INDEX IF EXISTS idx_payment_object;
DROP INDEX IF EXISTS idx_payment_user_initiated;
DROP INDEX IF EXISTS ux_payment_provider_txn;

DROP INDEX IF EXISTS idx_notifications_user_status;
DROP INDEX IF EXISTS idx_messages_chat_created;

DROP INDEX IF EXISTS idx_course_progress_executor_status;
DROP INDEX IF EXISTS ux_course_assignments_active_per_course_executor;
DROP INDEX IF EXISTS idx_course_assignments_executor_status;
DROP INDEX IF EXISTS idx_course_materials_course_sort;
DROP INDEX IF EXISTS idx_courses_coach;

DROP INDEX IF EXISTS idx_executor_rating_snapshots_executor;
DROP INDEX IF EXISTS idx_sanctions_executor_active;
DROP INDEX IF EXISTS idx_reviews_executor_created;

DROP INDEX IF EXISTS idx_response_status_history_response;
DROP INDEX IF EXISTS ux_responses_one_accepted_per_order;
DROP INDEX IF EXISTS idx_responses_executor;
DROP INDEX IF EXISTS idx_responses_order;

DROP INDEX IF EXISTS idx_order_status_history_order;
DROP INDEX IF EXISTS idx_order_promotions_order;
DROP INDEX IF EXISTS idx_orders_selected_executor;
DROP INDEX IF EXISTS idx_orders_client_history;
DROP INDEX IF EXISTS idx_orders_active_feed;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_selected_response;

DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chat_participants;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS course_progress;
DROP TABLE IF EXISTS course_assignments;
DROP TABLE IF EXISTS course_materials;
DROP TABLE IF EXISTS courses;
DROP TABLE IF EXISTS executor_rating_snapshots;
DROP TABLE IF EXISTS sanctions;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS response_status_history;
DROP TABLE IF EXISTS responses;
DROP TABLE IF EXISTS order_status_history;
DROP TABLE IF EXISTS order_promotions;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS coach_profiles;
DROP TABLE IF EXISTS executor_profiles;
DROP TABLE IF EXISTS client_profiles;

DROP TYPE IF EXISTS payment_object_type;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS notification_channel;
DROP TYPE IF EXISTS notification_status;
DROP TYPE IF EXISTS message_sender_type;
DROP TYPE IF EXISTS course_assignment_status;
DROP TYPE IF EXISTS sanction_status;
DROP TYPE IF EXISTS response_status;
DROP TYPE IF EXISTS order_status;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'audit_logs' AND column_name = 'metadata'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'audit_logs' AND column_name = 'payload'
    ) THEN
        ALTER TABLE audit_logs RENAME COLUMN metadata TO payload;
    END IF;
END $$;
