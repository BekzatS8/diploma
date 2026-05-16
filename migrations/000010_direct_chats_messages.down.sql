DROP INDEX IF EXISTS idx_messages_chat_created_active;

ALTER TABLE messages
    ALTER COLUMN body DROP DEFAULT;

DROP INDEX IF EXISTS idx_chats_user_b;
DROP INDEX IF EXISTS idx_chats_user_a;
DROP INDEX IF EXISTS ux_chats_direct_pair;
DROP INDEX IF EXISTS ux_chats_order_id;

ALTER TABLE chats DROP CONSTRAINT IF EXISTS chats_type_check;

DELETE FROM chats WHERE chat_type='direct';

ALTER TABLE chats
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS user_b_id,
    DROP COLUMN IF EXISTS user_a_id,
    DROP COLUMN IF EXISTS chat_type;

ALTER TABLE chats
    ALTER COLUMN order_id SET NOT NULL;

ALTER TABLE chats
    ADD CONSTRAINT chats_order_id_key UNIQUE(order_id);
