package chats

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
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
		ON CONFLICT (order_id) WHERE order_id IS NOT NULL DO NOTHING
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

func (r *Repository) EnsureDirectChat(ctx context.Context, userID, peerID string) (ChatDetail, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ChatDetail{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND is_active=TRUE)`, peerID).Scan(&exists); err != nil {
		return ChatDetail{}, err
	}
	if !exists {
		return ChatDetail{}, pgx.ErrNoRows
	}

	a, b := orderedPair(userID, peerID)
	var chatID string
	err = tx.QueryRow(ctx, `
		INSERT INTO chats(chat_type, user_a_id, user_b_id)
		VALUES('direct', $1, $2)
		ON CONFLICT (LEAST(user_a_id, user_b_id), GREATEST(user_a_id, user_b_id))
		WHERE chat_type='direct' AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL
		DO NOTHING
		RETURNING id
	`, a, b).Scan(&chatID)
	if err != nil {
		if err != pgx.ErrNoRows {
			return ChatDetail{}, err
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM chats WHERE chat_type='direct' AND LEAST(user_a_id,user_b_id)=LEAST($1::uuid,$2::uuid) AND GREATEST(user_a_id,user_b_id)=GREATEST($1::uuid,$2::uuid)`, userID, peerID).Scan(&chatID); err != nil {
			return ChatDetail{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO chat_participants(chat_id, user_id)
		VALUES($1,$2),($1,$3)
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`, chatID, userID, peerID); err != nil {
		return ChatDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ChatDetail{}, err
	}
	return r.getChatDetail(ctx, chatID)
}

func (r *Repository) ListMyChats(ctx context.Context, q ListChatsQuery) ([]ChatSummary, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM chat_participants WHERE user_id=$1`, q.UserID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			c.id,
			c.chat_type,
			c.order_id::text,
			c.user_a_id::text,
			c.user_b_id::text,
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
		if err := rows.Scan(&it.ChatID, &it.ChatType, &it.OrderID, &it.UserAID, &it.UserBID, &it.LastMessagePreview, &it.LastMessageAt, &it.UnreadCount, &participantsRaw); err != nil {
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
		SELECT c.id, c.chat_type, c.order_id::text, o.status, o.client_id::text, o.selected_executor_id::text,
			c.user_a_id::text, c.user_b_id::text,
			(
				SELECT COALESCE(MAX(m.created_at), NULL) FROM messages m WHERE m.chat_id=c.id AND m.deleted_at IS NULL
			),
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object('user_id', p.user_id, 'joined_at', p.joined_at, 'last_read_at', p.last_read_at) ORDER BY p.joined_at),'[]'::jsonb)
				FROM chat_participants p
				WHERE p.chat_id = c.id
			)
		FROM chats c
		LEFT JOIN orders o ON o.id = c.order_id
		WHERE c.id=$1 AND (o.id IS NULL OR o.deleted_at IS NULL)
	`, chatID)
	var item ChatDetail
	var participantsRaw []byte
	if err := row.Scan(&item.ChatID, &item.ChatType, &item.OrderID, &item.OrderStatus, &item.ClientID, &item.SelectedExecutorID, &item.UserAID, &item.UserBID, &item.LastMessageAt, &participantsRaw); err != nil {
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
		SELECT id, chat_id, sender_user_id, sender_type, body, created_at, edited_at, deleted_at
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
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderUserID, &m.SenderType, &m.Text, &m.CreatedAt, &m.EditedAt, &m.DeletedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.loadAttachments(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateMessageResult struct {
	Message     Message
	OrderID     *string
	ReceiverID  string
	ReceiverSet bool
}

func (r *Repository) CreateMessage(ctx context.Context, chatID, senderUserID, text string, uploadIDs []string) (CreateMessageResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CreateMessageResult{}, err
	}
	defer tx.Rollback(ctx)

	var orderID *string
	if err := tx.QueryRow(ctx, `SELECT order_id::text FROM chats WHERE id=$1 FOR UPDATE`, chatID).Scan(&orderID); err != nil {
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
		RETURNING id, chat_id, sender_user_id, sender_type, body, created_at, edited_at, deleted_at
	`, chatID, senderUserID, text).Scan(&msg.ID, &msg.ChatID, &msg.SenderUserID, &msg.SenderType, &msg.Text, &msg.CreatedAt, &msg.EditedAt, &msg.DeletedAt); err != nil {
		return CreateMessageResult{}, err
	}
	if err := r.attachUploadsTx(ctx, tx, msg.ID, chatID, senderUserID, uploadIDs); err != nil {
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
	items := []Message{msg}
	_ = r.loadAttachments(ctx, items)
	msg = items[0]
	return CreateMessageResult{Message: msg, OrderID: orderID, ReceiverID: receiverID, ReceiverSet: receiverSet}, nil
}

func (r *Repository) UpdateMessage(ctx context.Context, chatID, messageID, userID, text string) (Message, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE messages
		SET body=$4, edited_at=NOW()
		WHERE id=$1 AND chat_id=$2 AND sender_user_id=$3 AND deleted_at IS NULL
		RETURNING id, chat_id, sender_user_id, sender_type, body, created_at, edited_at, deleted_at
	`, messageID, chatID, userID, text)
	var msg Message
	if err := row.Scan(&msg.ID, &msg.ChatID, &msg.SenderUserID, &msg.SenderType, &msg.Text, &msg.CreatedAt, &msg.EditedAt, &msg.DeletedAt); err != nil {
		return Message{}, err
	}
	items := []Message{msg}
	if err := r.loadAttachments(ctx, items); err != nil {
		return Message{}, err
	}
	msg = items[0]
	return msg, nil
}

func (r *Repository) DeleteMessage(ctx context.Context, chatID, messageID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE messages
		SET deleted_at=NOW()
		WHERE id=$1 AND chat_id=$2 AND sender_user_id=$3 AND deleted_at IS NULL
	`, messageID, chatID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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
			c.chat_type,
			c.order_id::text,
			c.user_a_id::text,
			c.user_b_id::text,
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
		if err := rows.Scan(&it.ChatID, &it.ChatType, &it.OrderID, &it.UserAID, &it.UserBID, &it.LastMessagePreview, &it.LastMessageAt, &it.UnreadCount, &participantsRaw); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(participantsRaw, &it.Participants); err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (r *Repository) attachUploadsTx(ctx context.Context, tx pgx.Tx, messageID, chatID, userID string, uploadIDs []string) error {
	seen := map[string]struct{}{}
	sortOrder := 0
	for _, uploadID := range uploadIDs {
		uploadID = strings.TrimSpace(uploadID)
		if uploadID == "" {
			continue
		}
		if _, exists := seen[uploadID]; exists {
			continue
		}
		seen[uploadID] = struct{}{}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM uploads WHERE id=$1 AND author_id=$2)`, uploadID, userID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return pgx.ErrNoRows
		}
		metadata, _ := json.Marshal(map[string]any{"chat_id": chatID})
		if _, err := tx.Exec(ctx, `
			INSERT INTO attachments(id, upload_id, target_type, target_id, sort_order, metadata)
			VALUES($1, $2, 'chat_attachment', $3, $4, $5::jsonb)
			ON CONFLICT (upload_id, target_type, target_id) DO NOTHING
		`, uuid.NewString(), uploadID, messageID, sortOrder, string(metadata)); err != nil {
			return err
		}
		sortOrder++
	}
	return nil
}

func (r *Repository) loadAttachments(ctx context.Context, messages []Message) error {
	for i := range messages {
		rows, err := r.db.Query(ctx, `
			SELECT a.id::text, a.upload_id::text, u.file_path, u.original_name, u.mime_type, u.size_bytes, a.created_at
			FROM attachments a
			JOIN uploads u ON u.id=a.upload_id
			WHERE a.target_type='chat_attachment' AND a.target_id=$1::uuid
			ORDER BY a.sort_order ASC, a.created_at ASC
		`, messages[i].ID)
		if err != nil {
			return err
		}
		attachments := make([]MessageAttachment, 0)
		for rows.Next() {
			var item MessageAttachment
			if err := rows.Scan(&item.ID, &item.UploadID, &item.FilePath, &item.OriginalName, &item.MimeType, &item.SizeBytes, &item.CreatedAt); err != nil {
				rows.Close()
				return err
			}
			attachments = append(attachments, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		messages[i].Attachments = attachments
	}
	return nil
}

func orderedPair(a, b string) (string, string) {
	if strings.Compare(a, b) <= 0 {
		return a, b
	}
	return b, a
}

func (r *Repository) ListMessagesAdmin(ctx context.Context, chatID string, q ListMessagesQuery) ([]Message, int64, error) {
	if _, err := r.GetChatByIDAdmin(ctx, chatID); err != nil {
		return nil, 0, err
	}
	return r.ListMessages(ctx, chatID, q)
}
