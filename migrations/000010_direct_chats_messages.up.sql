ALTER TABLE chats
    ALTER COLUMN order_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS chat_type TEXT NOT NULL DEFAULT 'order',
    ADD COLUMN IF NOT EXISTS user_a_id UUID REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS user_b_id UUID REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE chats DROP CONSTRAINT IF EXISTS chats_order_id_key;
ALTER TABLE chats DROP CONSTRAINT IF EXISTS chats_type_check;
ALTER TABLE chats
    ADD CONSTRAINT chats_type_check CHECK (chat_type IN ('order', 'direct'));

UPDATE chats c
SET chat_type='order',
    user_a_id=p.client_id,
    user_b_id=p.executor_id
FROM (
    SELECT cp.chat_id,
           MAX(CASE WHEN u.role='client' THEN cp.user_id END) AS client_id,
           MAX(CASE WHEN u.role='executor' THEN cp.user_id END) AS executor_id
    FROM chat_participants cp
    JOIN users u ON u.id=cp.user_id
    GROUP BY cp.chat_id
) p
WHERE c.id=p.chat_id AND c.chat_type='order';

CREATE UNIQUE INDEX IF NOT EXISTS ux_chats_order_id
    ON chats(order_id)
    WHERE order_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_chats_direct_pair
    ON chats(LEAST(user_a_id, user_b_id), GREATEST(user_a_id, user_b_id))
    WHERE chat_type='direct' AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chats_user_a ON chats(user_a_id)
    WHERE user_a_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_chats_user_b ON chats(user_b_id)
    WHERE user_b_id IS NOT NULL;

ALTER TABLE messages
    ALTER COLUMN body SET DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_messages_chat_created_active
    ON messages(chat_id, created_at)
    WHERE deleted_at IS NULL;
