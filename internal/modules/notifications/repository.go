package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type CreateParams struct {
	UserID  string
	Type    string
	Channel string
	Status  string
	Payload map[string]any
}

type ListQuery struct {
	Page       int
	PageSize   int
	UserID     string
	Type       string
	Status     string
	Channel    string
	UnreadOnly bool
}

func (r *Repository) Create(ctx context.Context, p CreateParams) (Notification, error) {
	payload := p.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Notification{}, err
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO notifications(user_id, type, channel, status, payload, sent_at)
		VALUES($1,$2,$3::notification_channel,$4::notification_status,$5::jsonb, CASE WHEN $4::text IN ('sent','read') THEN NOW() ELSE NULL END)
		RETURNING id, user_id, type, channel, status, payload, created_at, sent_at, read_at
	`, p.UserID, p.Type, p.Channel, p.Status, body)
	var n Notification
	err = row.Scan(&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Status, &n.Payload, &n.CreatedAt, &n.SentAt, &n.ReadAt)
	return n, err
}

func (r *Repository) ListMy(ctx context.Context, userID string, q ListQuery) ([]Notification, int64, error) {
	q.UserID = userID
	return r.list(ctx, q, true)
}

func (r *Repository) ListAdmin(ctx context.Context, q ListQuery) ([]Notification, int64, error) {
	return r.list(ctx, q, false)
}

func (r *Repository) list(ctx context.Context, q ListQuery, enforceUser bool) ([]Notification, int64, error) {
	where := make([]string, 0, 5)
	args := make([]any, 0, 8)
	argPos := 1

	if enforceUser {
		where = append(where, fmt.Sprintf("user_id=$%d", argPos))
		args = append(args, q.UserID)
		argPos++
	} else if strings.TrimSpace(q.UserID) != "" {
		where = append(where, fmt.Sprintf("user_id=$%d", argPos))
		args = append(args, q.UserID)
		argPos++
	}
	if strings.TrimSpace(q.Type) != "" {
		where = append(where, fmt.Sprintf("type=$%d", argPos))
		args = append(args, q.Type)
		argPos++
	}
	if strings.TrimSpace(q.Status) != "" {
		where = append(where, fmt.Sprintf("status=$%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	if strings.TrimSpace(q.Channel) != "" {
		where = append(where, fmt.Sprintf("channel=$%d", argPos))
		args = append(args, q.Channel)
		argPos++
	}
	if q.UnreadOnly {
		where = append(where, "read_at IS NULL")
	}
	whereSQL := "1=1"
	if len(where) > 0 {
		whereSQL = strings.Join(where, " AND ")
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, type, channel, status, payload, created_at, sent_at, read_at
		FROM notifications
		WHERE `+whereSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $`+fmt.Sprintf("%d", argPos)+` OFFSET $`+fmt.Sprintf("%d", argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Notification, 0)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Status, &n.Payload, &n.CreatedAt, &n.SentAt, &n.ReadAt); err != nil {
			return nil, 0, err
		}
		items = append(items, n)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetMyByID(ctx context.Context, id, userID string) (Notification, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, channel, status, payload, created_at, sent_at, read_at
		FROM notifications
		WHERE id=$1 AND user_id=$2
	`, id, userID)
	var n Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Status, &n.Payload, &n.CreatedAt, &n.SentAt, &n.ReadAt)
	return n, err
}

func (r *Repository) GetByID(ctx context.Context, id string) (Notification, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, channel, status, payload, created_at, sent_at, read_at
		FROM notifications
		WHERE id=$1
	`, id)
	var n Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Status, &n.Payload, &n.CreatedAt, &n.SentAt, &n.ReadAt)
	return n, err
}

func (r *Repository) MarkRead(ctx context.Context, id, userID string) (Notification, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE notifications
		SET read_at = COALESCE(read_at, NOW()),
			status = CASE WHEN status <> 'read' THEN 'read' ELSE status END,
			sent_at = COALESCE(sent_at, NOW())
		WHERE id=$1 AND user_id=$2
		RETURNING id, user_id, type, channel, status, payload, created_at, sent_at, read_at
	`, id, userID)
	var n Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Status, &n.Payload, &n.CreatedAt, &n.SentAt, &n.ReadAt)
	return n, err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	cmd, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET read_at = COALESCE(read_at, NOW()),
			status = CASE WHEN status <> 'read' THEN 'read' ELSE status END,
			sent_at = COALESCE(sent_at, NOW())
		WHERE user_id=$1 AND read_at IS NULL
	`, userID)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}
