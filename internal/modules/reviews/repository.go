package reviews

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type Review struct {
	ID           string  `json:"id"`
	OrderID      string  `json:"order_id"`
	ClientID     string  `json:"client_id"`
	ExecutorID   string  `json:"executor_id"`
	ReviewerID   string  `json:"reviewer_id"`
	RevieweeID   string  `json:"reviewee_id"`
	RevieweeRole string  `json:"reviewee_role"`
	Direction    string  `json:"direction"`
	Rating       int     `json:"rating"`
	Comment      *string `json:"comment,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type CreateReviewParams struct {
	OrderID      string
	ClientID     string
	ExecutorID   string
	ReviewerID   string
	RevieweeID   string
	RevieweeRole string
	Direction    string
	Rating       int
	Comment      *string
}

type ListQuery struct{ Page, PageSize int }

type MyReview struct {
	ID           string          `json:"id"`
	Source       string          `json:"source"`
	OrderID      *string         `json:"order_id,omitempty"`
	TargetType   *string         `json:"target_type,omitempty"`
	TargetID     *string         `json:"target_id,omitempty"`
	RevieweeID   *string         `json:"reviewee_id,omitempty"`
	RevieweeRole *string         `json:"reviewee_role,omitempty"`
	Direction    *string         `json:"direction,omitempty"`
	Rating       int             `json:"rating"`
	Comment      *string         `json:"comment,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

type OrderReviewPreconditions struct {
	ClientID     string
	ExecutorID   string
	ReviewerID   string
	RevieweeID   string
	RevieweeRole string
	Direction    string
}

func (r *Repository) CanCreateReview(ctx context.Context, orderID, userID, role string) (OrderReviewPreconditions, bool, error) {
	var p OrderReviewPreconditions
	err := r.db.QueryRow(ctx, `
		SELECT o.client_id, resp.executor_id
		FROM orders o
		JOIN responses resp ON resp.id = o.selected_response_id
		WHERE o.id=$1 AND o.deleted_at IS NULL AND o.status='completed' AND o.selected_response_id IS NOT NULL
	`, orderID).Scan(&p.ClientID, &p.ExecutorID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return OrderReviewPreconditions{}, false, nil
		}
		return OrderReviewPreconditions{}, false, err
	}

	switch role {
	case "client":
		if !strings.EqualFold(p.ClientID, userID) {
			return OrderReviewPreconditions{}, false, nil
		}
		p.ReviewerID = p.ClientID
		p.RevieweeID = p.ExecutorID
		p.RevieweeRole = "executor"
		p.Direction = "client_to_executor"
	case "executor":
		if !strings.EqualFold(p.ExecutorID, userID) {
			return OrderReviewPreconditions{}, false, nil
		}
		p.ReviewerID = p.ExecutorID
		p.RevieweeID = p.ClientID
		p.RevieweeRole = "client"
		p.Direction = "executor_to_client"
	default:
		return OrderReviewPreconditions{}, false, nil
	}

	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM reviews
			WHERE order_id=$1 AND reviewer_id=$2 AND reviewee_id=$3 AND deleted_at IS NULL
		)
	`, orderID, p.ReviewerID, p.RevieweeID).Scan(&exists); err != nil {
		return OrderReviewPreconditions{}, false, err
	}
	return p, !exists, nil
}

func (r *Repository) Create(ctx context.Context, p CreateReviewParams) (Review, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO reviews(order_id, client_id, executor_id, reviewer_id, reviewee_id, reviewee_role, direction, rating, comment)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id::text, order_id::text, client_id::text, executor_id::text, reviewer_id::text, reviewee_id::text, reviewee_role, direction, rating, comment, created_at::text, updated_at::text
	`, p.OrderID, p.ClientID, p.ExecutorID, p.ReviewerID, p.RevieweeID, p.RevieweeRole, p.Direction, p.Rating, p.Comment)
	return scanReview(row)
}

func (r *Repository) CreateTx(ctx context.Context, tx pgx.Tx, p CreateReviewParams) (Review, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO reviews(order_id, client_id, executor_id, reviewer_id, reviewee_id, reviewee_role, direction, rating, comment)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id::text, order_id::text, client_id::text, executor_id::text, reviewer_id::text, reviewee_id::text, reviewee_role, direction, rating, comment, created_at::text, updated_at::text
	`, p.OrderID, p.ClientID, p.ExecutorID, p.ReviewerID, p.RevieweeID, p.RevieweeRole, p.Direction, p.Rating, p.Comment)
	return scanReview(row)
}

func (r *Repository) GetByOrder(ctx context.Context, orderID string) (Review, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id::text, order_id::text, client_id::text, executor_id::text, reviewer_id::text, reviewee_id::text, reviewee_role, direction, rating, comment, created_at::text, updated_at::text
		FROM reviews
		WHERE order_id=$1 AND direction='client_to_executor' AND deleted_at IS NULL
	`, orderID)
	return scanReview(row)
}

func (r *Repository) ListExecutor(ctx context.Context, executorID string, q ListQuery) ([]Review, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE reviewee_id=$1 AND reviewee_role='executor' AND deleted_at IS NULL`, executorID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, order_id::text, client_id::text, executor_id::text, reviewer_id::text, reviewee_id::text, reviewee_role, direction, rating, comment, created_at::text, updated_at::text
		FROM reviews WHERE reviewee_id=$1 AND reviewee_role='executor' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, executorID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Review, 0)
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, rv)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListUser(ctx context.Context, userID string, q ListQuery) ([]Review, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE reviewee_id=$1 AND deleted_at IS NULL`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, order_id::text, client_id::text, executor_id::text, reviewer_id::text, reviewee_id::text, reviewee_role, direction, rating, comment, created_at::text, updated_at::text
		FROM reviews
		WHERE reviewee_id=$1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Review, 0)
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, rv)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListAuthored(ctx context.Context, userID string, q ListQuery) ([]MyReview, int64, error) {
	var orderTotal int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE reviewer_id=$1 AND deleted_at IS NULL`, userID).Scan(&orderTotal); err != nil {
		return nil, 0, err
	}
	var entityTotal int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM entity_reviews WHERE author_id=$1 AND deleted_at IS NULL AND COALESCE(metadata->>'source','') <> 'course_review'`, userID).Scan(&entityTotal); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT source, id, order_id, target_type, target_id, reviewee_id, reviewee_role, direction, rating, comment, metadata, created_at, updated_at
		FROM (
			SELECT 'order'::text AS source,
			       id::text,
			       order_id::text,
			       NULL::text AS target_type,
			       reviewee_id::text AS target_id,
			       reviewee_id::text,
			       reviewee_role,
			       direction,
			       rating,
			       comment,
			       '{}'::jsonb AS metadata,
			       created_at,
			       updated_at
			FROM reviews
			WHERE reviewer_id=$1 AND deleted_at IS NULL
			UNION ALL
			SELECT 'entity'::text AS source,
			       id::text,
			       NULL::text AS order_id,
			       target_type,
			       target_id::text,
			       NULL::text AS reviewee_id,
			       NULL::text AS reviewee_role,
			       NULL::text AS direction,
			       rating,
			       comment,
			       metadata,
			       created_at,
			       updated_at
			FROM entity_reviews
			WHERE author_id=$1 AND deleted_at IS NULL AND COALESCE(metadata->>'source','') <> 'course_review'
		) authored
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]MyReview, 0)
	for rows.Next() {
		item, err := scanMyReview(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, orderTotal + entityTotal, rows.Err()
}

func (r *Repository) GetAuthored(ctx context.Context, id, userID string) (MyReview, error) {
	row := r.db.QueryRow(ctx, `
		SELECT source, id, order_id, target_type, target_id, reviewee_id, reviewee_role, direction, rating, comment, metadata, created_at, updated_at
		FROM (
			SELECT 'order'::text AS source,
			       id::text,
			       order_id::text,
			       NULL::text AS target_type,
			       reviewee_id::text AS target_id,
			       reviewee_id::text,
			       reviewee_role,
			       direction,
			       rating,
			       comment,
			       '{}'::jsonb AS metadata,
			       created_at,
			       updated_at
			FROM reviews
			WHERE id=$1 AND reviewer_id=$2 AND deleted_at IS NULL
			UNION ALL
			SELECT 'entity'::text AS source,
			       id::text,
			       NULL::text AS order_id,
			       target_type,
			       target_id::text,
			       NULL::text AS reviewee_id,
			       NULL::text AS reviewee_role,
			       NULL::text AS direction,
			       rating,
			       comment,
			       metadata,
			       created_at,
			       updated_at
			FROM entity_reviews
			WHERE id=$1 AND author_id=$2 AND deleted_at IS NULL AND COALESCE(metadata->>'source','') <> 'course_review'
		) authored
		LIMIT 1
	`, id, userID)
	return scanMyReview(row)
}

func (r *Repository) UpdateOrderReviewOwnedTx(ctx context.Context, tx pgx.Tx, id, userID string, rating int, comment *string) (Review, error) {
	row := tx.QueryRow(ctx, `
		UPDATE reviews
		SET rating=$3,
		    comment=$4,
		    updated_at=NOW()
		WHERE id=$1 AND reviewer_id=$2 AND deleted_at IS NULL
		RETURNING id::text, order_id::text, client_id::text, executor_id::text, reviewer_id::text, reviewee_id::text, reviewee_role, direction, rating, comment, created_at::text, updated_at::text
	`, id, userID, rating, comment)
	return scanReview(row)
}

func (r *Repository) DeleteOrderReviewOwnedTx(ctx context.Context, tx pgx.Tx, id, userID string) (Review, error) {
	row := tx.QueryRow(ctx, `
		UPDATE reviews
		SET deleted_at=NOW(),
		    updated_at=NOW()
		WHERE id=$1 AND reviewer_id=$2 AND deleted_at IS NULL
		RETURNING id::text, order_id::text, client_id::text, executor_id::text, reviewer_id::text, reviewee_id::text, reviewee_role, direction, rating, comment, created_at::text, updated_at::text
	`, id, userID)
	return scanReview(row)
}

func (r *Repository) UpdateMirroredOrderEntityReviewsTx(ctx context.Context, tx pgx.Tx, reviewID string, rating int, comment *string) error {
	_, err := tx.Exec(ctx, `
		UPDATE entity_reviews
		SET rating=$2,
		    comment=$3,
		    updated_at=NOW()
		WHERE metadata->>'legacy_review_id'=$1
		  AND deleted_at IS NULL
	`, reviewID, rating, comment)
	return err
}

func (r *Repository) DeleteMirroredOrderEntityReviewsTx(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE entity_reviews
		SET deleted_at=NOW(),
		    updated_at=NOW()
		WHERE metadata->>'legacy_review_id'=$1
		  AND deleted_at IS NULL
	`, reviewID)
	return err
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

func scanReview(row interface{ Scan(dest ...any) error }) (Review, error) {
	var rv Review
	err := row.Scan(&rv.ID, &rv.OrderID, &rv.ClientID, &rv.ExecutorID, &rv.ReviewerID, &rv.RevieweeID, &rv.RevieweeRole, &rv.Direction, &rv.Rating, &rv.Comment, &rv.CreatedAt, &rv.UpdatedAt)
	return rv, err
}

func scanMyReview(row interface{ Scan(dest ...any) error }) (MyReview, error) {
	var item MyReview
	var metadata []byte
	err := row.Scan(&item.Source, &item.ID, &item.OrderID, &item.TargetType, &item.TargetID, &item.RevieweeID, &item.RevieweeRole, &item.Direction, &item.Rating, &item.Comment, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return MyReview{}, err
	}
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	item.Metadata = json.RawMessage(metadata)
	return item, nil
}

func (r *Repository) CanCreateCourseReview(ctx context.Context, userID, role, courseID string) (bool, error) {
	if role != "executor" {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM course_assignments ca
			JOIN courses c ON c.id = ca.course_id
			WHERE ca.course_id=$1::uuid
			  AND ca.executor_id=$2::uuid
			  AND ca.status='completed'
			  AND c.deleted_at IS NULL
			  AND NOT EXISTS (
			  	SELECT 1 FROM entity_reviews er
			  	WHERE er.author_id=$2::uuid
			  	  AND er.target_type='course'
			  	  AND er.target_id=$1::uuid
			  	  AND er.deleted_at IS NULL
			  )
		)
	`, courseID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetCourseOwner(ctx context.Context, courseID string) (string, bool, error) {
	var ownerID string
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(created_by, coach_id)::text
		FROM courses
		WHERE id=$1 AND deleted_at IS NULL
	`, courseID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return ownerID, ownerID != "", nil
}
