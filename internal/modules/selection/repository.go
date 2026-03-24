package selection

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type Selection struct {
	OrderID                string  `json:"order_id"`
	OrderStatus            string  `json:"order_status"`
	SelectedResponseID     *string `json:"selected_response_id,omitempty"`
	SelectedExecutorID     *string `json:"selected_executor_id,omitempty"`
	SelectedResponseStatus *string `json:"selected_response_status,omitempty"`
}

func (r *Repository) SelectResponse(ctx context.Context, orderID, responseID, actorID string) (string, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)

	var orderStatus string
	var selectedResponseID *string
	var selectedExecutorID *string
	var clientID string
	if err := tx.QueryRow(ctx, `SELECT status, selected_response_id, selected_executor_id, client_id FROM orders WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, orderID).Scan(&orderStatus, &selectedResponseID, &selectedExecutorID, &clientID); err != nil {
		return "", false, err
	}

	var targetStatus string
	var isPaid bool
	var executorID string
	if err := tx.QueryRow(ctx, `SELECT status, is_paid, executor_id FROM responses WHERE id=$1 AND order_id=$2 AND deleted_at IS NULL FOR UPDATE`, responseID, orderID).Scan(&targetStatus, &isPaid, &executorID); err != nil {
		return "", false, err
	}

	if orderStatus == "in_progress" && selectedResponseID != nil && *selectedResponseID == responseID && targetStatus == "accepted" {
		return executorID, false, tx.Commit(ctx)
	}
	if orderStatus != "published" {
		return "", false, ErrInvalidState
	}
	if targetStatus != "submitted" || !isPaid {
		return "", false, ErrInvalidState
	}
	if selectedResponseID != nil {
		return "", false, ErrAlreadySelected
	}

	if _, err := tx.Exec(ctx, `
		UPDATE orders
		SET selected_response_id=$2, selected_executor_id=$3, status='in_progress', updated_at=NOW()
		WHERE id=$1
	`, orderID, responseID, executorID); err != nil {
		return "", false, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO order_status_history(order_id, old_status, new_status, changed_by, reason) VALUES($1,$2,'in_progress',$3,$4)`, orderID, orderStatus, actorID, "response selected"); err != nil {
		return "", false, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE responses SET status='accepted', updated_at=NOW() WHERE id=$1
	`, responseID); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO response_status_history(response_id, old_status, new_status, changed_by, reason) VALUES($1,'submitted','accepted',$2,$3)`, responseID, actorID, "selected by client"); err != nil {
		return "", false, err
	}

	rows, err := tx.Query(ctx, `SELECT id FROM responses WHERE order_id=$1 AND id<>$2 AND deleted_at IS NULL AND status='submitted' FOR UPDATE`, orderID, responseID)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()
	otherIDs := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", false, err
		}
		otherIDs = append(otherIDs, id)
	}
	if rows.Err() != nil {
		return "", false, rows.Err()
	}
	for _, id := range otherIDs {
		if _, err := tx.Exec(ctx, `UPDATE responses SET status='rejected', updated_at=NOW() WHERE id=$1`, id); err != nil {
			return "", false, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO response_status_history(response_id, old_status, new_status, changed_by, reason) VALUES($1,'submitted','rejected',$2,$3)`, id, actorID, "another response selected"); err != nil {
			return "", false, err
		}
	}

	var chatID string
	chatErr := tx.QueryRow(ctx, `
		INSERT INTO chats(order_id)
		VALUES($1)
		ON CONFLICT (order_id) DO NOTHING
		RETURNING id
	`, orderID).Scan(&chatID)
	if chatErr != nil {
		if chatErr != pgx.ErrNoRows {
			return "", false, chatErr
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM chats WHERE order_id=$1`, orderID).Scan(&chatID); err != nil {
			return "", false, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO chat_participants(chat_id, user_id)
		VALUES($1,$2),($1,$3)
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`, chatID, clientID, executorID); err != nil {
		return "", false, err
	}

	return executorID, true, tx.Commit(ctx)
}

func (r *Repository) GetSelection(ctx context.Context, orderID string) (Selection, error) {
	row := r.db.QueryRow(ctx, `
		SELECT o.id, o.status, o.selected_response_id, o.selected_executor_id, r.status
		FROM orders o
		LEFT JOIN responses r ON r.id = o.selected_response_id
		WHERE o.id=$1 AND o.deleted_at IS NULL
	`, orderID)
	var s Selection
	if err := row.Scan(&s.OrderID, &s.OrderStatus, &s.SelectedResponseID, &s.SelectedExecutorID, &s.SelectedResponseStatus); err != nil {
		return Selection{}, err
	}
	return s, nil
}

func (r *Repository) Complete(ctx context.Context, orderID, actorID string) (string, error) {
	return r.transitionOrder(ctx, orderID, actorID, "in_progress", "completed", "completed by client")
}

func (r *Repository) Reopen(ctx context.Context, orderID, actorID string) error {
	_, err := r.transitionOrder(ctx, orderID, actorID, "completed", "in_progress", "reopened by client")
	return err
}

func (r *Repository) transitionOrder(ctx context.Context, orderID, actorID, fromStatus, toStatus, reason string) (string, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var current string
	var selectedResponseID *string
	var selectedExecutorID *string
	if err := tx.QueryRow(ctx, `SELECT status, selected_response_id, selected_executor_id FROM orders WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, orderID).Scan(&current, &selectedResponseID, &selectedExecutorID); err != nil {
		return "", err
	}
	if current != fromStatus {
		return "", ErrInvalidState
	}
	if toStatus == "completed" && selectedResponseID == nil {
		return "", ErrInvalidState
	}
	if toStatus == "in_progress" {
		var hasReview bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE order_id=$1 AND deleted_at IS NULL)`, orderID).Scan(&hasReview); err != nil {
			return "", err
		}
		if hasReview {
			return "", ErrInvalidState
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status=$2, updated_at=NOW(), completed_at=CASE WHEN $2='completed' THEN NOW() ELSE completed_at END WHERE id=$1`, orderID, toStatus); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO order_status_history(order_id, old_status, new_status, changed_by, reason) VALUES($1,$2,$3,$4,$5)`, orderID, current, toStatus, actorID, reason); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if selectedExecutorID == nil {
		return "", nil
	}
	return *selectedExecutorID, nil
}

func (r *Repository) IsOrderOwnerOrAdmin(ctx context.Context, orderID, userID, role string) (bool, error) {
	if role == "admin" {
		return true, nil
	}
	if role != "client" {
		return false, nil
	}
	var clientID string
	err := r.db.QueryRow(ctx, `SELECT client_id FROM orders WHERE id=$1 AND deleted_at IS NULL`, orderID).Scan(&clientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, pgx.ErrNoRows
		}
		return false, err
	}
	return clientID == userID, nil
}
