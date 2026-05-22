package attachments

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type orderAttachmentTarget struct {
	ClientID           string
	SelectedExecutorID *string
}

type responseAttachmentTarget struct {
	ExecutorID    string
	OrderClientID string
	Status        string
	IsPaid        bool
}

type courseAttachmentTarget struct {
	CoachID   *string
	CreatedBy *string
	Status    string
}

type reviewAttachmentTarget struct {
	AuthorID *string
}

func (r *Repository) GetOrderTarget(ctx context.Context, targetID string) (orderAttachmentTarget, error) {
	row := r.db.QueryRow(ctx, `
		SELECT client_id::text, selected_executor_id::text
		FROM orders
		WHERE id=$1::uuid AND deleted_at IS NULL
	`, targetID)
	var item orderAttachmentTarget
	err := row.Scan(&item.ClientID, &item.SelectedExecutorID)
	return item, mapNoRows(err)
}

func (r *Repository) GetResponseTarget(ctx context.Context, targetID string) (responseAttachmentTarget, error) {
	row := r.db.QueryRow(ctx, `
		SELECT r.executor_id::text, o.client_id::text, r.status::text, r.is_paid
		FROM responses r
		JOIN orders o ON o.id = r.order_id
		WHERE r.id=$1::uuid AND r.deleted_at IS NULL AND o.deleted_at IS NULL
	`, targetID)
	var item responseAttachmentTarget
	err := row.Scan(&item.ExecutorID, &item.OrderClientID, &item.Status, &item.IsPaid)
	return item, mapNoRows(err)
}

func (r *Repository) ChatTargetExists(ctx context.Context, targetID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM chats c WHERE c.id=$1::uuid)
		    OR EXISTS(SELECT 1 FROM messages m WHERE m.id=$1::uuid AND m.deleted_at IS NULL)
	`, targetID).Scan(&exists)
	return exists, err
}

func (r *Repository) IsChatTargetParticipant(ctx context.Context, targetID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM chat_participants cp
			WHERE cp.user_id=$2::uuid
			  AND (
			      cp.chat_id=$1::uuid
			      OR cp.chat_id IN (
			          SELECT m.chat_id
			          FROM messages m
			          WHERE m.id=$1::uuid AND m.deleted_at IS NULL
			      )
			  )
		)
	`, targetID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetCourseTarget(ctx context.Context, targetID string) (courseAttachmentTarget, error) {
	row := r.db.QueryRow(ctx, `
		SELECT coach_id::text, created_by::text, status
		FROM courses
		WHERE id=$1::uuid AND deleted_at IS NULL
	`, targetID)
	var item courseAttachmentTarget
	err := row.Scan(&item.CoachID, &item.CreatedBy, &item.Status)
	return item, mapNoRows(err)
}

func (r *Repository) UserExists(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1::uuid)`, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetReviewTarget(ctx context.Context, targetID string) (reviewAttachmentTarget, error) {
	row := r.db.QueryRow(ctx, `
		SELECT author_id
		FROM (
			SELECT author_id::text AS author_id
			FROM entity_reviews
			WHERE id=$1::uuid AND deleted_at IS NULL
			UNION ALL
			SELECT client_id::text AS author_id
			FROM reviews
			WHERE id=$1::uuid AND deleted_at IS NULL
		) review_targets
		LIMIT 1
	`, targetID)
	var item reviewAttachmentTarget
	err := row.Scan(&item.AuthorID)
	return item, mapNoRows(err)
}

func mapNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
