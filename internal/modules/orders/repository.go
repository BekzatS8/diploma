package orders

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

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type CreateOrderParams struct {
	ClientID     string
	CategoryID   *int64
	Title        string
	Description  string
	BudgetAmount float64
	Currency     string
}

type UpdateOrderParams struct {
	Title        *string
	Description  *string
	CategoryID   *int64
	BudgetAmount *float64
	Currency     *string
}

func (r *Repository) ResolveCategoryIDBySlug(ctx context.Context, slug string) (*int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1 AND is_active = TRUE`, slug).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

func (r *Repository) Create(ctx context.Context, p CreateOrderParams) (Order, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO orders (client_id, category_id, title, description, budget_amount, currency, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'draft')
		RETURNING id, client_id, category_id, title, description, budget_amount, currency, status,
		       selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
	`, p.ClientID, p.CategoryID, p.Title, p.Description, p.BudgetAmount, p.Currency)
	return scanOrder(row)
}

func (r *Repository) GetByID(ctx context.Context, id string) (Order, error) {
	row := r.db.QueryRow(ctx, `
		SELECT o.id, o.client_id, o.category_id, c.slug, c.name, o.title, o.description, o.budget_amount, o.currency,
		       o.status, o.selected_executor_id, o.published_at, o.completed_at, o.cancelled_at,
		       o.created_at, o.updated_at, o.deleted_at
		FROM orders o
		LEFT JOIN categories c ON c.id = o.category_id
		WHERE o.id = $1
	`, id)
	return scanOrderWithCategory(row)
}

func (r *Repository) GetMyByID(ctx context.Context, id, clientID string) (Order, error) {
	row := r.db.QueryRow(ctx, `
		SELECT o.id, o.client_id, o.category_id, c.slug, c.name, o.title, o.description, o.budget_amount, o.currency,
		       o.status, o.selected_executor_id, o.published_at, o.completed_at, o.cancelled_at,
		       o.created_at, o.updated_at, o.deleted_at
		FROM orders o
		LEFT JOIN categories c ON c.id = o.category_id
		WHERE o.id = $1 AND o.client_id = $2 AND o.deleted_at IS NULL
	`, id, clientID)
	return scanOrderWithCategory(row)
}

func (r *Repository) ListMy(ctx context.Context, clientID string, q MyOrdersQuery) ([]Order, int64, error) {
	where := []string{"o.client_id = $1", "o.deleted_at IS NULL"}
	args := []any{clientID}
	argPos := 2
	if q.Status != "" {
		where = append(where, fmt.Sprintf("o.status = $%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	whereSQL := strings.Join(where, " AND ")

	countSQL := `SELECT COUNT(*) FROM orders o WHERE ` + whereSQL
	var total int64
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	listSQL := fmt.Sprintf(`
		SELECT o.id, o.client_id, o.category_id, c.slug, c.name, o.title, o.description, o.budget_amount, o.currency,
		       o.status, o.selected_executor_id, o.published_at, o.completed_at, o.cancelled_at,
		       o.created_at, o.updated_at, o.deleted_at
		FROM orders o
		LEFT JOIN categories c ON c.id = o.category_id
		WHERE %s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argPos, argPos+1)

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Order, 0)
	for rows.Next() {
		item, err := scanOrderWithCategory(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListPublic(ctx context.Context, q PublicOrdersQuery) ([]Order, int64, error) {
	where := []string{"o.deleted_at IS NULL", "o.status = 'published'"}
	args := make([]any, 0)
	argPos := 1

	if q.CategorySlug != "" {
		where = append(where, fmt.Sprintf("c.slug = $%d", argPos))
		args = append(args, q.CategorySlug)
		argPos++
	}
	if q.BudgetMin != nil {
		where = append(where, fmt.Sprintf("o.budget_amount >= $%d", argPos))
		args = append(args, *q.BudgetMin)
		argPos++
	}
	if q.BudgetMax != nil {
		where = append(where, fmt.Sprintf("o.budget_amount <= $%d", argPos))
		args = append(args, *q.BudgetMax)
		argPos++
	}
	if q.Q != "" {
		where = append(where, fmt.Sprintf("(o.title ILIKE $%d OR o.description ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+q.Q+"%")
		argPos++
	}
	whereSQL := strings.Join(where, " AND ")

	countSQL := `
		SELECT COUNT(*)
		FROM orders o
		LEFT JOIN categories c ON c.id = o.category_id
		WHERE ` + whereSQL
	var total int64
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	listSQL := fmt.Sprintf(`
		SELECT o.id, o.client_id, o.category_id, c.slug, c.name, o.title, o.description, o.budget_amount, o.currency,
		       o.status, o.selected_executor_id, o.published_at, o.completed_at, o.cancelled_at,
		       o.created_at, o.updated_at, o.deleted_at
		FROM orders o
		LEFT JOIN categories c ON c.id = o.category_id
		WHERE %s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argPos, argPos+1)
	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Order, 0)
	for rows.Next() {
		item, err := scanOrderWithCategory(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) UpdateDraft(ctx context.Context, id, clientID string, p UpdateOrderParams) (Order, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE orders
		SET title = COALESCE($3, title),
		    description = COALESCE($4, description),
		    category_id = COALESCE($5, category_id),
		    budget_amount = COALESCE($6, budget_amount),
		    currency = COALESCE($7, currency),
		    updated_at = NOW()
		WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL
		RETURNING id, client_id, category_id, title, description, budget_amount, currency, status,
		       selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
	`, id, clientID, p.Title, p.Description, p.CategoryID, p.BudgetAmount, p.Currency)
	return scanOrder(row)
}

func (r *Repository) SoftDelete(ctx context.Context, id, clientID string) error {
	_, err := r.db.Exec(ctx, `UPDATE orders SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL`, id, clientID)
	return err
}

func (r *Repository) LatestPaymentByOrderID(ctx context.Context, orderID string) (*PaymentTransaction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, order_id, provider, provider_transaction_id, amount, currency, status, initiated_at
		FROM payment_transactions
		WHERE order_id = $1
		ORDER BY initiated_at DESC
		LIMIT 1
	`, orderID)
	var t PaymentTransaction
	if err := row.Scan(&t.ID, &t.OrderID, &t.Provider, &t.ProviderTransactionID, &t.Amount, &t.Currency, &t.Status, &t.InitiatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *Repository) SubmitWithPayment(ctx context.Context, orderID, clientID string, amount float64, currency, provider string, chargeProviderRef string, chargeCheckoutURL string) (Order, PaymentTransaction, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Order{}, PaymentTransaction{}, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var selectedExecutorID *string
	currentRow := tx.QueryRow(ctx, `
		SELECT status, selected_executor_id
		FROM orders
		WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`, orderID, clientID)
	if err := currentRow.Scan(&currentStatus, &selectedExecutorID); err != nil {
		return Order{}, PaymentTransaction{}, err
	}

	if currentStatus != "draft" {
		return Order{}, PaymentTransaction{}, ErrInvalidStatusTransition
	}

	row := tx.QueryRow(ctx, `
		UPDATE orders
		SET status = 'payment_pending', updated_at = NOW()
		WHERE id = $1
		RETURNING id, client_id, category_id, title, description, budget_amount, currency, status,
		       selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
	`, orderID)
	updated, err := scanOrder(row)
	if err != nil {
		return Order{}, PaymentTransaction{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_history (order_id, old_status, new_status, changed_by, reason)
		VALUES ($1, $2, 'payment_pending', $3, $4)
	`, orderID, currentStatus, clientID, "order submitted"); err != nil {
		return Order{}, PaymentTransaction{}, err
	}

	payRow := tx.QueryRow(ctx, `
		INSERT INTO payment_transactions (
			user_id, object_type, object_id, order_id, provider, provider_transaction_id,
			amount, currency, status, metadata
		)
		VALUES ($1, 'order_posting', $2, $2, $3, $4, $5, $6, 'pending', $7::jsonb)
		RETURNING id, order_id, provider, provider_transaction_id, amount, currency, status, initiated_at
	`, clientID, orderID, provider, chargeProviderRef, amount, currency,
		fmt.Sprintf(`{"checkout_url":%q}`, chargeCheckoutURL),
	)
	var payment PaymentTransaction
	if err := payRow.Scan(&payment.ID, &payment.OrderID, &payment.Provider, &payment.ProviderTransactionID, &payment.Amount, &payment.Currency, &payment.Status, &payment.InitiatedAt); err != nil {
		return Order{}, PaymentTransaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, PaymentTransaction{}, err
	}
	return updated, payment, nil
}

func (r *Repository) Cancel(ctx context.Context, orderID, clientID, reason string) (Order, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var selectedExecutorID *string
	if err := tx.QueryRow(ctx, `
		SELECT status, selected_executor_id
		FROM orders
		WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`, orderID, clientID).Scan(&currentStatus, &selectedExecutorID); err != nil {
		return Order{}, err
	}

	if currentStatus != "draft" && currentStatus != "payment_pending" && currentStatus != "published" {
		return Order{}, ErrInvalidStatusTransition
	}
	if currentStatus == "published" && selectedExecutorID != nil {
		return Order{}, ErrInvalidStatusTransition
	}

	row := tx.QueryRow(ctx, `
		UPDATE orders
		SET status='cancelled', cancelled_at = NOW(), updated_at = NOW()
		WHERE id = $1
		RETURNING id, client_id, category_id, title, description, budget_amount, currency, status,
		       selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
	`, orderID)
	updated, err := scanOrder(row)
	if err != nil {
		return Order{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_history (order_id, old_status, new_status, changed_by, reason)
		VALUES ($1, $2, 'cancelled', $3, $4)
	`, orderID, currentStatus, clientID, reason); err != nil {
		return Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return updated, nil
}

func scanOrder(row pgx.Row) (Order, error) {
	var o Order
	err := row.Scan(
		&o.ID,
		&o.ClientID,
		&o.CategoryID,
		&o.Title,
		&o.Description,
		&o.BudgetAmount,
		&o.Currency,
		&o.Status,
		&o.SelectedExecutorID,
		&o.PublishedAt,
		&o.CompletedAt,
		&o.CancelledAt,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.DeletedAt,
	)
	return o, err
}

func scanOrderWithCategory(row interface{ Scan(dest ...any) error }) (Order, error) {
	var o Order
	err := row.Scan(
		&o.ID,
		&o.ClientID,
		&o.CategoryID,
		&o.CategorySlug,
		&o.CategoryName,
		&o.Title,
		&o.Description,
		&o.BudgetAmount,
		&o.Currency,
		&o.Status,
		&o.SelectedExecutorID,
		&o.PublishedAt,
		&o.CompletedAt,
		&o.CancelledAt,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.DeletedAt,
	)
	return o, err
}
