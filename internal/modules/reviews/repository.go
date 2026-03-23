package reviews

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type Review struct {
	ID         string  `json:"id"`
	OrderID    string  `json:"order_id"`
	ClientID   string  `json:"client_id"`
	ExecutorID string  `json:"executor_id"`
	Rating     int     `json:"rating"`
	Comment    *string `json:"comment,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type CreateReviewParams struct {
	OrderID    string
	ClientID   string
	ExecutorID string
	Rating     int
	Comment    *string
}

type ListQuery struct{ Page, PageSize int }

func (r *Repository) CanCreateReview(ctx context.Context, orderID string) (clientID string, executorID string, ok bool, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT o.client_id, resp.executor_id
		FROM orders o
		JOIN responses resp ON resp.id = o.selected_response_id
		LEFT JOIN reviews rv ON rv.order_id = o.id AND rv.deleted_at IS NULL
		WHERE o.id=$1 AND o.deleted_at IS NULL AND o.status='completed' AND o.selected_response_id IS NOT NULL AND rv.id IS NULL
	`, orderID).Scan(&clientID, &executorID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", false, nil
		}
		return "", "", false, err
	}
	return clientID, executorID, true, nil
}

func (r *Repository) Create(ctx context.Context, p CreateReviewParams) (Review, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO reviews(order_id, client_id, executor_id, rating, comment)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id, order_id, client_id, executor_id, rating, comment, created_at::text, updated_at::text
	`, p.OrderID, p.ClientID, p.ExecutorID, p.Rating, p.Comment)
	var rv Review
	err := row.Scan(&rv.ID, &rv.OrderID, &rv.ClientID, &rv.ExecutorID, &rv.Rating, &rv.Comment, &rv.CreatedAt, &rv.UpdatedAt)
	return rv, err
}

func (r *Repository) CreateTx(ctx context.Context, tx pgx.Tx, p CreateReviewParams) (Review, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO reviews(order_id, client_id, executor_id, rating, comment)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id, order_id, client_id, executor_id, rating, comment, created_at::text, updated_at::text
	`, p.OrderID, p.ClientID, p.ExecutorID, p.Rating, p.Comment)
	var rv Review
	err := row.Scan(&rv.ID, &rv.OrderID, &rv.ClientID, &rv.ExecutorID, &rv.Rating, &rv.Comment, &rv.CreatedAt, &rv.UpdatedAt)
	return rv, err
}

func (r *Repository) GetByOrder(ctx context.Context, orderID string) (Review, error) {
	row := r.db.QueryRow(ctx, `SELECT id, order_id, client_id, executor_id, rating, comment, created_at::text, updated_at::text FROM reviews WHERE order_id=$1 AND deleted_at IS NULL`, orderID)
	var rv Review
	err := row.Scan(&rv.ID, &rv.OrderID, &rv.ClientID, &rv.ExecutorID, &rv.Rating, &rv.Comment, &rv.CreatedAt, &rv.UpdatedAt)
	return rv, err
}

func (r *Repository) ListExecutor(ctx context.Context, executorID string, q ListQuery) ([]Review, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE executor_id=$1 AND deleted_at IS NULL`, executorID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, client_id, executor_id, rating, comment, created_at::text, updated_at::text
		FROM reviews WHERE executor_id=$1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, executorID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Review, 0)
	for rows.Next() {
		var rv Review
		if err := rows.Scan(&rv.ID, &rv.OrderID, &rv.ClientID, &rv.ExecutorID, &rv.Rating, &rv.Comment, &rv.CreatedAt, &rv.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, rv)
	}
	return items, total, rows.Err()
}

func (r *Repository) IsOwnerOrAdmin(ctx context.Context, orderID, userID, role string) (bool, error) {
	if role == "admin" {
		return true, nil
	}
	if role != "client" {
		return false, nil
	}
	var clientID string
	err := r.db.QueryRow(ctx, `SELECT client_id FROM orders WHERE id=$1 AND deleted_at IS NULL`, orderID).Scan(&clientID)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(clientID, userID), nil
}
