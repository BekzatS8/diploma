package chats

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) EnsureChatForSelectionTx(ctx context.Context, tx pgx.Tx, orderID, clientID, executorID string) (string, error) {
	var chatID string
	err := tx.QueryRow(ctx, `
		INSERT INTO chats(order_id)
		VALUES($1)
		ON CONFLICT (order_id) DO NOTHING
		RETURNING id
	`, orderID).Scan(&chatID)
	if err != nil {
		if err != pgx.ErrNoRows {
			return "", err
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM chats WHERE order_id=$1`, orderID).Scan(&chatID); err != nil {
			return "", err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO chat_participants(chat_id, user_id)
		VALUES($1,$2),($1,$3)
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`, chatID, clientID, executorID); err != nil {
		return "", err
	}
	return chatID, nil
}

func (r *Repository) ListMyChats(ctx context.Context, q ListChatsQuery) ([]ChatSummary, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM chat_participants WHERE user_id=$1`, q.UserID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			c.id,
			c.order_id,
			lm.body,
			lm.created_at,
			COALESCE((
				SELECT COUNT(*) FROM messages mu
				WHERE mu.chat_id=c.id AND mu.deleted_at IS NULL
				  AND mu.created_at > COALESCE(cp.last_read_at, '-infinity'::timestamptz)
				  AND (mu.sender_user_id IS NULL OR mu.sender_user_id <> $1)
			), 0) AS unread_count,
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('user_id', p.user_id, 'joined_at', p.joined_at, 'last_read_at', p.last_read_at) ORDER BY p.joined_at),'[]'::jsonb)
				FROM chat_participants p
				WHERE p.chat_id = c.id
			)
		FROM chat_participants cp
		JOIN chats c ON c.id = cp.chat_id
		LEFT JOIN LATERAL (
			SELECT body, created_at
			FROM messages m
			WHERE m.chat_id = c.id AND m.deleted_at IS NULL
			ORDER BY m.created_at DESC
			LIMIT 1
		) lm ON true
		WHERE cp.user_id=$1
		ORDER BY COALESCE(lm.created_at, c.created_at) DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`, q.UserID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]ChatSummary, 0)
	for rows.Next() {
		var it ChatSummary
		var participantsRaw []byte
		if err := rows.Scan(&it.ChatID, &it.OrderID, &it.LastMessagePreview, &it.LastMessageAt, &it.UnreadCount, &participantsRaw); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(participantsRaw, &it.Participants); err != nil {
			return nil, 0, err
		}
		it.HasUnread = it.UnreadCount > 0
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetMyChatByID(ctx context.Context, chatID, userID string) (ChatDetail, error) {
	if ok, err := r.IsParticipant(ctx, chatID, userID); err != nil {
		return ChatDetail{}, err
	} else if !ok {
		return ChatDetail{}, pgx.ErrNoRows
	}
	return r.getChatDetail(ctx, chatID)
}

func (r *Repository) GetChatByIDAdmin(ctx context.Context, chatID string) (ChatDetail, error) {
	return r.getChatDetail(ctx, chatID)
}

func (r *Repository) getChatDetail(ctx context.Context, chatID string) (ChatDetail, error) {
	row := r.db.QueryRow(ctx, `
		SELECT c.id, c.order_id, o.status, o.client_id, o.selected_executor_id,
			(
				SELECT COALESCE(MAX(m.created_at), NULL) FROM messages m WHERE m.chat_id=c.id AND m.deleted_at IS NULL
			),
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('user_id', p.user_id, 'joined_at', p.joined_at, 'last_read_at', p.last_read_at) ORDER BY p.joined_at),'[]'::jsonb)
				FROM chat_participants p
				WHERE p.chat_id = c.id
			)
		FROM chats c
		JOIN orders o ON o.id = c.order_id
		WHERE c.id=$1 AND o.deleted_at IS NULL
	`, chatID)
	var item ChatDetail
	var participantsRaw []byte
	if err := row.Scan(&item.ChatID, &item.OrderID, &item.OrderStatus, &item.ClientID, &item.SelectedExecutorID, &item.LastMessageAt, &participantsRaw); err != nil {
		return ChatDetail{}, err
	}
	if err := json.Unmarshal(participantsRaw, &item.Participants); err != nil {
		return ChatDetail{}, err
	}
	return item, nil
}

func (r *Repository) IsParticipant(ctx context.Context, chatID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id=$1 AND user_id=$2)`, chatID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) ListMessages(ctx context.Context, chatID string, q ListMessagesQuery) ([]Message, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE chat_id=$1 AND deleted_at IS NULL`, chatID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, chat_id, sender_user_id, sender_type, body, created_at
		FROM messages
		WHERE chat_id=$1 AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3
	`, chatID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderUserID, &m.SenderType, &m.Text, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	return items, total, rows.Err()
}

type CreateMessageResult struct {
	Message     Message
	OrderID     string
	ReceiverID  string
	ReceiverSet bool
}

func (r *Repository) CreateMessage(ctx context.Context, chatID, senderUserID, text string) (CreateMessageResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CreateMessageResult{}, err
	}
	defer tx.Rollback(ctx)

	var orderID string
	if err := tx.QueryRow(ctx, `SELECT order_id FROM chats WHERE id=$1 FOR UPDATE`, chatID).Scan(&orderID); err != nil {
		return CreateMessageResult{}, err
	}

	var isParticipant bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id=$1 AND user_id=$2)`, chatID, senderUserID).Scan(&isParticipant); err != nil {
		return CreateMessageResult{}, err
	}
	if !isParticipant {
		return CreateMessageResult{}, pgx.ErrNoRows
	}

	var msg Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages(chat_id, sender_user_id, sender_type, body)
		VALUES($1,$2,'user',$3)
		RETURNING id, chat_id, sender_user_id, sender_type, body, created_at
	`, chatID, senderUserID, text).Scan(&msg.ID, &msg.ChatID, &msg.SenderUserID, &msg.SenderType, &msg.Text, &msg.CreatedAt); err != nil {
		return CreateMessageResult{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE chat_participants SET last_read_at=GREATEST(COALESCE(last_read_at, '-infinity'::timestamptz), $3) WHERE chat_id=$1 AND user_id=$2`, chatID, senderUserID, msg.CreatedAt); err != nil {
		return CreateMessageResult{}, err
	}

	var receiverID string
	err = tx.QueryRow(ctx, `SELECT user_id FROM chat_participants WHERE chat_id=$1 AND user_id<>$2 ORDER BY joined_at ASC LIMIT 1`, chatID, senderUserID).Scan(&receiverID)
	receiverSet := true
	if err != nil {
		if err == pgx.ErrNoRows {
			receiverSet = false
		} else {
			return CreateMessageResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CreateMessageResult{}, err
	}
	return CreateMessageResult{Message: msg, OrderID: orderID, ReceiverID: receiverID, ReceiverSet: receiverSet}, nil
}

func (r *Repository) MarkRead(ctx context.Context, chatID, userID string) error {
	cmd, err := r.db.Exec(ctx, `UPDATE chat_participants SET last_read_at=NOW() WHERE chat_id=$1 AND user_id=$2`, chatID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListAdminChats(ctx context.Context, page, pageSize int) ([]ChatSummary, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM chats`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			c.id,
			c.order_id,
			lm.body,
			lm.created_at,
			0::bigint,
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('user_id', p.user_id, 'joined_at', p.joined_at, 'last_read_at', p.last_read_at) ORDER BY p.joined_at),'[]'::jsonb)
				FROM chat_participants p
				WHERE p.chat_id = c.id
			)
		FROM chats c
		LEFT JOIN LATERAL (
			SELECT body, created_at FROM messages m WHERE m.chat_id=c.id AND m.deleted_at IS NULL ORDER BY m.created_at DESC LIMIT 1
		) lm ON true
		ORDER BY COALESCE(lm.created_at, c.created_at) DESC, c.id DESC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]ChatSummary, 0)
	for rows.Next() {
		var it ChatSummary
		var participantsRaw []byte
		if err := rows.Scan(&it.ChatID, &it.OrderID, &it.LastMessagePreview, &it.LastMessageAt, &it.UnreadCount, &participantsRaw); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(participantsRaw, &it.Participants); err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListMessagesAdmin(ctx context.Context, chatID string, q ListMessagesQuery) ([]Message, int64, error) {
	if _, err := r.GetChatByIDAdmin(ctx, chatID); err != nil {
		return nil, 0, err
	}
	return r.ListMessages(ctx, chatID, q)
}
