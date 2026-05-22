package devpayments

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type Transaction struct {
	ID         string
	Status     string
	ObjectType string
	OrderID    *string
	ResponseID *string
	UserID     string
	PaidAt     *time.Time
}

type ConfirmResult struct {
	TransactionID           string
	OrderPublishedForClient *OrderPublishedEvent
	ResponseSubmittedClient *ResponseSubmittedEvent
}

type OrderPublishedEvent struct {
	OrderID  string
	ClientID string
}

type ResponseSubmittedEvent struct {
	ResponseID string
	OrderID    string
	ClientID   string
}

func (r *Repository) GetTransaction(ctx context.Context, tx pgx.Tx, id string) (Transaction, error) {
	row := tx.QueryRow(ctx, `SELECT id::text, status, object_type, order_id::text, response_id::text, user_id::text, paid_at FROM payment_transactions WHERE id=$1 FOR UPDATE`, id)
	var t Transaction
	err := row.Scan(&t.ID, &t.Status, &t.ObjectType, &t.OrderID, &t.ResponseID, &t.UserID, &t.PaidAt)
	return t, err
}

func (r *Repository) Confirm(ctx context.Context, id string) (ConfirmResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ConfirmResult{}, err
	}
	defer tx.Rollback(ctx)
	result := ConfirmResult{TransactionID: id}

	payment, err := r.GetTransaction(ctx, tx, id)
	if err != nil {
		return ConfirmResult{}, err
	}
	if payment.Status == "succeeded" {
		return result, tx.Commit(ctx)
	}
	if payment.Status != "pending" {
		return ConfirmResult{}, fmt.Errorf("cannot confirm payment with status %s", payment.Status)
	}

	if _, err := tx.Exec(ctx, `UPDATE payment_transactions SET status='succeeded', paid_at=NOW() WHERE id=$1`, id); err != nil {
		return ConfirmResult{}, err
	}

	switch payment.ObjectType {
	case "order_posting":
		if payment.OrderID == nil {
			return ConfirmResult{}, fmt.Errorf("order id is required")
		}
		var old string
		if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1 FOR UPDATE`, *payment.OrderID).Scan(&old); err != nil {
			return ConfirmResult{}, err
		}
		if old == "payment_pending" {
			if _, err := tx.Exec(ctx, `UPDATE orders SET status='published', published_at=NOW(), updated_at=NOW() WHERE id=$1`, *payment.OrderID); err != nil {
				return ConfirmResult{}, err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO order_status_history(order_id, old_status, new_status, changed_by, reason) VALUES($1,$2,'published',$3,$4)`, *payment.OrderID, old, payment.UserID, "payment confirmed (dev)"); err != nil {
				return ConfirmResult{}, err
			}
			result.OrderPublishedForClient = &OrderPublishedEvent{OrderID: *payment.OrderID, ClientID: payment.UserID}
		}
	case "response_submission":
		if payment.ResponseID == nil {
			return ConfirmResult{}, fmt.Errorf("response id is required")
		}
		var orderID string
		var old string
		if err := tx.QueryRow(ctx, `SELECT status, order_id FROM responses WHERE id=$1 FOR UPDATE`, *payment.ResponseID).Scan(&old, &orderID); err != nil {
			return ConfirmResult{}, err
		}
		if old == "payment_pending" {
			if _, err := tx.Exec(ctx, `UPDATE responses SET status='submitted', is_paid=TRUE, paid_at=NOW(), updated_at=NOW() WHERE id=$1`, *payment.ResponseID); err != nil {
				return ConfirmResult{}, err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO response_status_history(response_id, old_status, new_status, changed_by, reason) VALUES($1,$2,'submitted',$3,$4)`, *payment.ResponseID, old, payment.UserID, "payment confirmed (dev)"); err != nil {
				return ConfirmResult{}, err
			}
			result.ResponseSubmittedClient = &ResponseSubmittedEvent{ResponseID: *payment.ResponseID, OrderID: orderID, ClientID: payment.UserID}
		}
	}
	return result, tx.Commit(ctx)
}

func (r *Repository) Fail(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	payment, err := r.GetTransaction(ctx, tx, id)
	if err != nil {
		return err
	}
	if payment.Status == "failed" {
		return tx.Commit(ctx)
	}
	if payment.Status == "succeeded" {
		return fmt.Errorf("cannot fail succeeded payment")
	}
	if payment.Status != "pending" {
		return fmt.Errorf("cannot fail payment with status %s", payment.Status)
	}

	if _, err := tx.Exec(ctx, `UPDATE payment_transactions SET status='failed', failed_at=NOW() WHERE id=$1`, id); err != nil {
		return err
	}

	switch payment.ObjectType {
	case "order_posting":
		if payment.OrderID == nil {
			return fmt.Errorf("order id is required")
		}
		var old string
		if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1 FOR UPDATE`, *payment.OrderID).Scan(&old); err != nil {
			return err
		}
		if old == "payment_pending" {
			if _, err := tx.Exec(ctx, `UPDATE orders SET status='draft', updated_at=NOW() WHERE id=$1`, *payment.OrderID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO order_status_history(order_id, old_status, new_status, changed_by, reason) VALUES($1,$2,'draft',$3,$4)`, *payment.OrderID, old, payment.UserID, "payment failed (dev)"); err != nil {
				return err
			}
		}
	case "response_submission":
		if payment.ResponseID == nil {
			return fmt.Errorf("response id is required")
		}
		var old string
		if err := tx.QueryRow(ctx, `SELECT status FROM responses WHERE id=$1 FOR UPDATE`, *payment.ResponseID).Scan(&old); err != nil {
			return err
		}
		if old == "payment_pending" {
			if _, err := tx.Exec(ctx, `UPDATE responses SET status='draft', updated_at=NOW() WHERE id=$1`, *payment.ResponseID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO response_status_history(response_id, old_status, new_status, changed_by, reason) VALUES($1,$2,'draft',$3,$4)`, *payment.ResponseID, old, payment.UserID, "payment failed (dev)"); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}
