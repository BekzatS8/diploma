package responses

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type CreateParams struct {
	OrderID        string
	ExecutorID     string
	CoverLetter    *string
	ProposedAmount *float64
	Currency       string
}

type UpdateParams struct {
	CoverLetter    *string
	ProposedAmount *float64
	Currency       *string
}

func (r *Repository) GetOrderForResponse(ctx context.Context, orderID string) (clientID string, status string, deleted bool, err error) {
	var deletedAt any
	err = r.db.QueryRow(ctx, `SELECT client_id, status, deleted_at FROM orders WHERE id=$1`, orderID).Scan(&clientID, &status, &deletedAt)
	if err != nil {
		return "", "", false, err
	}
	deleted = deletedAt != nil
	return
}

func (r *Repository) Create(ctx context.Context, p CreateParams) (Response, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO responses (order_id, executor_id, cover_letter, proposed_amount, currency, status, is_paid)
		VALUES ($1,$2,$3,$4,$5,'draft',FALSE)
		RETURNING id, order_id, executor_id, cover_letter, proposed_amount, currency, status, is_paid, paid_at, created_at, updated_at, deleted_at
	`, p.OrderID, p.ExecutorID, p.CoverLetter, p.ProposedAmount, p.Currency)
	return scanResponse(row)
}

func (r *Repository) GetMyByID(ctx context.Context, orderID, responseID, executorID string) (Response, error) {
	row := r.db.QueryRow(ctx, `
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r
		JOIN orders o ON o.id = r.order_id
		WHERE r.id=$1 AND r.order_id=$2 AND r.executor_id=$3 AND r.deleted_at IS NULL
	`, responseID, orderID, executorID)
	return scanResponseWithOrder(row)
}

func (r *Repository) UpdateDraft(ctx context.Context, orderID, responseID, executorID string, p UpdateParams) (Response, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE responses
		SET cover_letter = COALESCE($4, cover_letter),
		    proposed_amount = COALESCE($5, proposed_amount),
		    currency = COALESCE($6, currency),
		    updated_at = NOW()
		WHERE id=$1 AND order_id=$2 AND executor_id=$3 AND deleted_at IS NULL
		RETURNING id, order_id, executor_id, cover_letter, proposed_amount, currency, status, is_paid, paid_at, created_at, updated_at, deleted_at
	`, responseID, orderID, executorID, p.CoverLetter, p.ProposedAmount, p.Currency)
	return scanResponse(row)
}

func (r *Repository) SoftDelete(ctx context.Context, orderID, responseID, executorID string) error {
	_, err := r.db.Exec(ctx, `UPDATE responses SET deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND order_id=$2 AND executor_id=$3 AND deleted_at IS NULL`, responseID, orderID, executorID)
	return err
}

func (r *Repository) ListExecutor(ctx context.Context, executorID string, q ListQuery) ([]Response, int64, error) {
	where := []string{"r.executor_id=$1", "r.deleted_at IS NULL"}
	args := []any{executorID}
	argPos := 2
	if q.Status != "" {
		where = append(where, fmt.Sprintf("r.status=$%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM responses r WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r
		JOIN orders o ON o.id=r.order_id
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argPos, argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Response, 0)
	for rows.Next() {
		it, er := scanResponseWithOrder(rows)
		if er != nil {
			return nil, 0, er
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListExecutorByOrder(ctx context.Context, orderID, executorID string, q ListQuery) ([]Response, int64, error) {
	where := []string{"r.order_id=$1", "r.executor_id=$2", "r.deleted_at IS NULL"}
	args := []any{orderID, executorID}
	argPos := 3
	if q.Status != "" {
		where = append(where, fmt.Sprintf("r.status=$%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM responses r WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r JOIN orders o ON o.id=r.order_id
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argPos, argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Response, 0)
	for rows.Next() {
		it, er := scanResponseWithOrder(rows)
		if er != nil {
			return nil, 0, er
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListAll(ctx context.Context, q ListQuery) ([]Response, int64, error) {
	where := []string{"r.deleted_at IS NULL"}
	args := []any{}
	argPos := 1
	if q.Status != "" {
		where = append(where, fmt.Sprintf("r.status=$%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM responses r WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r JOIN orders o ON o.id=r.order_id
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argPos, argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Response, 0)
	for rows.Next() {
		it, er := scanResponseWithOrder(rows)
		if er != nil {
			return nil, 0, er
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}
func (r *Repository) GetExecutorByID(ctx context.Context, responseID, executorID string) (Response, error) {
	row := r.db.QueryRow(ctx, `
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r
		JOIN orders o ON o.id = r.order_id
		WHERE r.id=$1 AND r.executor_id=$2 AND r.deleted_at IS NULL
	`, responseID, executorID)
	return scanResponseWithOrder(row)
}

func (r *Repository) GetByID(ctx context.Context, id string) (Response, error) {
	row := r.db.QueryRow(ctx, `
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r
		JOIN orders o ON o.id = r.order_id
		WHERE r.id=$1
	`, id)
	return scanResponseWithOrder(row)
}

func (r *Repository) ListForClientOrder(ctx context.Context, orderID string, q ListQuery) ([]Response, int64, error) {
	where := []string{"r.order_id=$1", "r.deleted_at IS NULL", "r.status IN ('submitted','accepted')", "r.is_paid=TRUE"}
	args := []any{orderID}
	argPos := 2
	if q.Status != "" {
		where = append(where, fmt.Sprintf("r.status=$%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM responses r WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r JOIN orders o ON o.id=r.order_id
		WHERE %s ORDER BY r.created_at DESC LIMIT $%d OFFSET $%d
	`, whereSQL, argPos, argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Response, 0)
	for rows.Next() {
		it, er := scanResponseWithOrder(rows)
		if er != nil {
			return nil, 0, er
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetClientOrderResponse(ctx context.Context, orderID, responseID string) (Response, error) {
	row := r.db.QueryRow(ctx, `
		SELECT r.id, r.order_id, r.executor_id, r.cover_letter, r.proposed_amount, r.currency, r.status, r.is_paid, r.paid_at,
		       r.created_at, r.updated_at, r.deleted_at, o.client_id, o.status, o.title
		FROM responses r JOIN orders o ON o.id=r.order_id
		WHERE r.order_id=$1 AND r.id=$2 AND r.deleted_at IS NULL AND r.status IN ('submitted','accepted') AND r.is_paid=TRUE
	`, orderID, responseID)
	return scanResponseWithOrder(row)
}

func (r *Repository) SubmitWithPayment(ctx context.Context, orderID, responseID, executorID string, amount float64, currency, provider, providerRef, checkoutURL string) (Response, PaymentTransaction, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Response{}, PaymentTransaction{}, err
	}
	defer tx.Rollback(ctx)

	var currentStatus, orderStatus string
	if err := tx.QueryRow(ctx, `
		SELECT r.status, o.status
		FROM responses r JOIN orders o ON o.id=r.order_id
		WHERE r.id=$1 AND r.order_id=$2 AND r.executor_id=$3 AND r.deleted_at IS NULL
		FOR UPDATE
	`, responseID, orderID, executorID).Scan(&currentStatus, &orderStatus); err != nil {
		return Response{}, PaymentTransaction{}, err
	}
	if !CanTransition(currentStatus, StatusPaymentPending) || orderStatus != "published" {
		return Response{}, PaymentTransaction{}, ErrInvalidStatus
	}

	row := tx.QueryRow(ctx, `
		UPDATE responses SET status='payment_pending', updated_at=NOW()
		WHERE id=$1
		RETURNING id, order_id, executor_id, cover_letter, proposed_amount, currency, status, is_paid, paid_at, created_at, updated_at, deleted_at
	`, responseID)
	updated, err := scanResponse(row)
	if err != nil {
		return Response{}, PaymentTransaction{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO response_status_history (response_id, old_status, new_status, changed_by, reason)
		VALUES ($1,$2,'payment_pending',$3,$4)
	`, responseID, currentStatus, executorID, "response submitted for payment"); err != nil {
		return Response{}, PaymentTransaction{}, err
	}

	pRow := tx.QueryRow(ctx, `
		INSERT INTO payment_transactions (user_id, object_type, object_id, response_id, provider, provider_transaction_id, yookassa_payment_id, amount, currency, status, metadata)
		VALUES ($1,'response_submission',$2,$2,$3,$4,CASE WHEN $3 = 'yookassa' THEN $4 ELSE NULL END,$5,$6,'pending',$7::jsonb)
		RETURNING id::text, response_id::text, provider, provider_transaction_id, amount, currency, status, initiated_at
	`, executorID, responseID, provider, providerRef, amount, currency, fmt.Sprintf(`{"checkout_url":%q}`, checkoutURL))
	var pay PaymentTransaction
	if err := pRow.Scan(&pay.ID, &pay.ResponseID, &pay.Provider, &pay.ProviderTransactionID, &pay.Amount, &pay.Currency, &pay.Status, &pay.InitiatedAt); err != nil {
		return Response{}, PaymentTransaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Response{}, PaymentTransaction{}, err
	}
	return updated, pay, nil
}

func (r *Repository) Cancel(ctx context.Context, orderID, responseID, executorID, reason string) (Response, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Response{}, err
	}
	defer tx.Rollback(ctx)
	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM responses WHERE id=$1 AND order_id=$2 AND executor_id=$3 AND deleted_at IS NULL FOR UPDATE`, responseID, orderID, executorID).Scan(&current); err != nil {
		return Response{}, err
	}
	if !CanCancel(current) {
		return Response{}, ErrInvalidStatus
	}
	row := tx.QueryRow(ctx, `UPDATE responses SET status='cancelled', updated_at=NOW() WHERE id=$1 RETURNING id, order_id, executor_id, cover_letter, proposed_amount, currency, status, is_paid, paid_at, created_at, updated_at, deleted_at`, responseID)
	updated, err := scanResponse(row)
	if err != nil {
		return Response{}, err
	}
	if current == StatusPaymentPending {
		if _, err := tx.Exec(ctx, `
			UPDATE payment_transactions
			SET status='cancelled', failed_at=NOW()
			WHERE response_id=$1 AND object_type='response_submission' AND status='pending'
		`, responseID); err != nil {
			return Response{}, err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO response_status_history (response_id, old_status, new_status, changed_by, reason) VALUES ($1,$2,'cancelled',$3,$4)`, responseID, current, executorID, reason); err != nil {
		return Response{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Response{}, err
	}
	return updated, nil
}

func scanResponse(row pgx.Row) (Response, error) {
	var r Response
	err := row.Scan(&r.ID, &r.OrderID, &r.ExecutorID, &r.CoverLetter, &r.ProposedAmount, &r.Currency, &r.Status, &r.IsPaid, &r.PaidAt, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt)
	return r, err
}

func scanResponseWithOrder(row interface{ Scan(dest ...any) error }) (Response, error) {
	var r Response
	err := row.Scan(&r.ID, &r.OrderID, &r.ExecutorID, &r.CoverLetter, &r.ProposedAmount, &r.Currency, &r.Status, &r.IsPaid, &r.PaidAt, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt, &r.OrderClientID, &r.OrderStatus, &r.OrderTitle)
	return r, err
}
