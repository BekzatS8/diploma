package orders

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	DeadlineAt   *time.Time
	Region       *string
	Promotions   []string
}

type UpdateOrderParams struct {
	Title        *string
	Description  *string
	CategoryID   *int64
	BudgetAmount *float64
	Currency     *string
	DeadlineAt   *time.Time
	Region       *string
	Promotions   []string
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
	promotionsJSON, _ := json.Marshal(p.Promotions)
	row := r.db.QueryRow(ctx, `
		INSERT INTO orders (client_id, category_id, title, description, budget_amount, currency, deadline_at, region, promotion_options, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, 'draft')
		RETURNING id, client_id, category_id, title, description, budget_amount, currency,
		       deadline_at, region, promotion_options, posting_fee, promotion_fee, escrow_amount, total_charge, payment_status,
		       promoted_until, pinned_until, highlighted_until, executor_paid_at,
		       status, selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
	`, p.ClientID, p.CategoryID, p.Title, p.Description, p.BudgetAmount, p.Currency, p.DeadlineAt, p.Region, string(promotionsJSON))
	return scanOrder(row)
}

func (r *Repository) GetByID(ctx context.Context, id string) (Order, error) {
	row := r.db.QueryRow(ctx, `
		SELECT o.id, o.client_id, o.category_id, c.slug, c.name, o.title, o.description, o.budget_amount, o.currency,
		       o.deadline_at, o.region, o.promotion_options, o.posting_fee, o.promotion_fee, o.escrow_amount, o.total_charge, o.payment_status,
		       o.promoted_until, o.pinned_until, o.highlighted_until, o.executor_paid_at,
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
		       o.deadline_at, o.region, o.promotion_options, o.posting_fee, o.promotion_fee, o.escrow_amount, o.total_charge, o.payment_status,
		       o.promoted_until, o.pinned_until, o.highlighted_until, o.executor_paid_at,
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
		       o.deadline_at, o.region, o.promotion_options, o.posting_fee, o.promotion_fee, o.escrow_amount, o.total_charge, o.payment_status,
		       o.promoted_until, o.pinned_until, o.highlighted_until, o.executor_paid_at,
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
	if q.Region != "" {
		where = append(where, fmt.Sprintf("(o.region = $%d OR o.region = 'online')", argPos))
		args = append(args, q.Region)
		argPos++
	}
	if q.DeadlineBefore != nil {
		where = append(where, fmt.Sprintf("(o.deadline_at IS NULL OR o.deadline_at <= $%d)", argPos))
		args = append(args, *q.DeadlineBefore)
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
		       o.deadline_at, o.region, o.promotion_options, o.posting_fee, o.promotion_fee, o.escrow_amount, o.total_charge, o.payment_status,
		       o.promoted_until, o.pinned_until, o.highlighted_until, o.executor_paid_at,
		       o.status, o.selected_executor_id, o.published_at, o.completed_at, o.cancelled_at,
		       o.created_at, o.updated_at, o.deleted_at
		FROM orders o
		LEFT JOIN categories c ON c.id = o.category_id
		WHERE %s
		ORDER BY (o.pinned_until IS NOT NULL AND o.pinned_until > NOW()) DESC,
		         (o.promoted_until IS NOT NULL AND o.promoted_until > NOW()) DESC,
		         o.created_at DESC
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
	var promotionsJSON *string
	if p.Promotions != nil {
		raw, _ := json.Marshal(p.Promotions)
		value := string(raw)
		promotionsJSON = &value
	}
	row := r.db.QueryRow(ctx, `
		UPDATE orders
		SET title = COALESCE($3, title),
		    description = COALESCE($4, description),
		    category_id = COALESCE($5, category_id),
		    budget_amount = COALESCE($6, budget_amount),
		    currency = COALESCE($7, currency),
		    deadline_at = COALESCE($8, deadline_at),
		    region = COALESCE($9, region),
		    promotion_options = COALESCE($10::jsonb, promotion_options),
		    updated_at = NOW()
		WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL
		RETURNING id, client_id, category_id, title, description, budget_amount, currency,
		       deadline_at, region, promotion_options, posting_fee, promotion_fee, escrow_amount, total_charge, payment_status,
		       promoted_until, pinned_until, highlighted_until, executor_paid_at,
		       status, selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
	`, id, clientID, p.Title, p.Description, p.CategoryID, p.BudgetAmount, p.Currency, p.DeadlineAt, p.Region, promotionsJSON)
	return scanOrder(row)
}

func (r *Repository) SoftDelete(ctx context.Context, id, clientID string) error {
	_, err := r.db.Exec(ctx, `UPDATE orders SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND client_id = $2 AND deleted_at IS NULL`, id, clientID)
	return err
}

func (r *Repository) LatestPaymentByOrderID(ctx context.Context, orderID string) (*PaymentTransaction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id::text, order_id::text, provider, provider_transaction_id, amount, currency, status, initiated_at
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

func (r *Repository) SubmitWithWallet(ctx context.Context, orderID, clientID string, postingFee, promotionFee, escrowAmount, totalCharge float64, currency string, promotions []string) (Order, PaymentTransaction, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Order{}, PaymentTransaction{}, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var selectedExecutorID *string
	var walletBalance float64
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
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets(user_id, balance, currency)
		VALUES($1, 0, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, clientID, currency); err != nil {
		return Order{}, PaymentTransaction{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id=$1 FOR UPDATE`, clientID).Scan(&walletBalance); err != nil {
		return Order{}, PaymentTransaction{}, err
	}
	if walletBalance < totalCharge {
		return Order{}, PaymentTransaction{}, ErrInsufficientBalance
	}

	var promotedUntil, pinnedUntil, highlightedUntil any
	now := time.Now()
	for _, p := range promotions {
		switch p {
		case "top", "promotion_top", "raise_top":
			promotedUntil = now.Add(72 * time.Hour)
		case "pin", "pinned", "promotion_pin":
			pinnedUntil = now.Add(30 * 24 * time.Hour)
		case "highlight", "highlighted", "promotion_highlight":
			highlightedUntil = now.Add(30 * 24 * time.Hour)
		}
	}

	row := tx.QueryRow(ctx, `
		UPDATE orders
		SET status = 'published',
		    published_at = COALESCE(published_at, NOW()),
		    posting_fee = $2,
		    promotion_fee = $3,
		    escrow_amount = $4,
		    total_charge = $5,
		    payment_status = 'paid',
		    promoted_until = $6,
		    pinned_until = $7,
		    highlighted_until = $8,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, client_id, category_id, title, description, budget_amount, currency,
		       deadline_at, region, promotion_options, posting_fee, promotion_fee, escrow_amount, total_charge, payment_status,
		       promoted_until, pinned_until, highlighted_until, executor_paid_at,
		       status, selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
	`, orderID, postingFee, promotionFee, escrowAmount, totalCharge, promotedUntil, pinnedUntil, highlightedUntil)
	updated, err := scanOrder(row)
	if err != nil {
		return Order{}, PaymentTransaction{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_history (order_id, old_status, new_status, changed_by, reason)
		VALUES ($1, $2, 'published', $3, $4)
	`, orderID, currentStatus, clientID, "paid from internal wallet"); err != nil {
		return Order{}, PaymentTransaction{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE wallets
		SET balance=balance-$2, updated_at=NOW()
		WHERE user_id=$1
	`, clientID, totalCharge); err != nil {
		return Order{}, PaymentTransaction{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_transactions(user_id, amount, direction, currency, reason, order_id, created_by, metadata)
		VALUES($1, $2, 'debit', $3, 'order_submit', $4, $1, jsonb_build_object('posting_fee',$5::numeric,'promotion_fee',$6::numeric,'escrow_amount',$7::numeric))
	`, clientID, totalCharge, currency, orderID, postingFee, promotionFee, escrowAmount); err != nil {
		return Order{}, PaymentTransaction{}, err
	}

	payRow := tx.QueryRow(ctx, `
		INSERT INTO payment_transactions (
			user_id, object_type, object_id, order_id, provider, provider_transaction_id,
			amount, currency, status, paid_at, metadata
		)
		VALUES ($1, 'order_posting', $2, $2, 'internal_wallet', $3, $4, $5, 'succeeded', NOW(), $6::jsonb)
		RETURNING id::text, order_id::text, provider, provider_transaction_id, amount, currency, status, initiated_at
	`, clientID, orderID, "wallet-"+orderID, totalCharge, currency,
		fmt.Sprintf(`{"posting_fee":%v,"promotion_fee":%v,"escrow_amount":%v}`, postingFee, promotionFee, escrowAmount),
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
		RETURNING id, client_id, category_id, title, description, budget_amount, currency,
		       deadline_at, region, promotion_options, posting_fee, promotion_fee, escrow_amount, total_charge, payment_status,
		       promoted_until, pinned_until, highlighted_until, executor_paid_at,
		       status, selected_executor_id, published_at, completed_at, cancelled_at, created_at, updated_at, deleted_at
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
	var promotions []byte
	err := row.Scan(
		&o.ID,
		&o.ClientID,
		&o.CategoryID,
		&o.Title,
		&o.Description,
		&o.BudgetAmount,
		&o.Currency,
		&o.DeadlineAt,
		&o.Region,
		&promotions,
		&o.PostingFee,
		&o.PromotionFee,
		&o.EscrowAmount,
		&o.TotalCharge,
		&o.PaymentStatus,
		&o.PromotedUntil,
		&o.PinnedUntil,
		&o.HighlightedUntil,
		&o.ExecutorPaidAt,
		&o.Status,
		&o.SelectedExecutorID,
		&o.PublishedAt,
		&o.CompletedAt,
		&o.CancelledAt,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.DeletedAt,
	)
	if len(promotions) > 0 {
		_ = json.Unmarshal(promotions, &o.PromotionOptions)
	}
	if o.PromotionOptions == nil {
		o.PromotionOptions = []string{}
	}
	return o, err
}

func scanOrderWithCategory(row interface{ Scan(dest ...any) error }) (Order, error) {
	var o Order
	var promotions []byte
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
		&o.DeadlineAt,
		&o.Region,
		&promotions,
		&o.PostingFee,
		&o.PromotionFee,
		&o.EscrowAmount,
		&o.TotalCharge,
		&o.PaymentStatus,
		&o.PromotedUntil,
		&o.PinnedUntil,
		&o.HighlightedUntil,
		&o.ExecutorPaidAt,
		&o.Status,
		&o.SelectedExecutorID,
		&o.PublishedAt,
		&o.CompletedAt,
		&o.CancelledAt,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.DeletedAt,
	)
	if len(promotions) > 0 {
		_ = json.Unmarshal(promotions, &o.PromotionOptions)
	}
	if o.PromotionOptions == nil {
		o.PromotionOptions = []string{}
	}
	return o, err
}
