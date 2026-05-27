package reviews

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

type EntityReview struct {
	ID         string          `json:"id"`
	AuthorID   *string         `json:"author_id,omitempty"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Rating     int             `json:"rating"`
	Comment    *string         `json:"comment,omitempty"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
}

type RatingSummary struct {
	TargetType  string  `json:"target_type"`
	TargetID    string  `json:"target_id"`
	RatingAvg   float64 `json:"rating_avg"`
	RatingCount int     `json:"rating_count"`
	UpdatedAt   *string `json:"updated_at,omitempty"`
}

type CreateEntityReviewParams struct {
	AuthorID   string
	TargetType string
	TargetID   string
	Rating     int
	Comment    *string
	Metadata   map[string]any
}

func (r *Repository) CreateEntityTx(ctx context.Context, tx pgx.Tx, p CreateEntityReviewParams) (EntityReview, error) {
	metadata, err := json.Marshal(p.Metadata)
	if err != nil {
		return EntityReview{}, err
	}
	if len(metadata) == 0 || string(metadata) == "null" {
		metadata = []byte("{}")
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO entity_reviews(author_id, target_type, target_id, rating, comment, metadata)
		VALUES($1,$2,$3,$4,$5,$6::jsonb)
		RETURNING id::text, author_id::text, target_type, target_id::text, rating, comment, metadata, created_at::text, updated_at::text
	`, p.AuthorID, p.TargetType, p.TargetID, p.Rating, p.Comment, string(metadata))

	return scanEntityReview(row)
}

func (r *Repository) RecalculateEntityRatingTx(ctx context.Context, tx pgx.Tx, targetType, targetID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO entity_rating_summaries(target_type, target_id, rating_avg, rating_count, updated_at)
		SELECT $1,
		       $2::uuid,
		       COALESCE(ROUND(AVG(rating)::numeric, 2), 0),
		       COUNT(*)::int,
		       NOW()
		FROM entity_reviews
		WHERE target_type=$1 AND target_id=$2::uuid AND deleted_at IS NULL
		ON CONFLICT (target_type, target_id) DO UPDATE
		SET rating_avg=EXCLUDED.rating_avg,
		    rating_count=EXCLUDED.rating_count,
		    updated_at=NOW()
	`, targetType, targetID)
	if err != nil {
		return err
	}
	if targetType == "course" {
		_, err = tx.Exec(ctx, `
			UPDATE courses c
			SET rating_avg=s.rating_avg,
			    rating_count=s.rating_count,
			    updated_at=NOW()
			FROM entity_rating_summaries s
			WHERE c.id=s.target_id
			  AND s.target_type='course'
			  AND c.id=$1::uuid
		`, targetID)
	}
	return err
}

func (r *Repository) ListByTarget(ctx context.Context, targetType, targetID string, q ListQuery) ([]EntityReview, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM entity_reviews
		WHERE target_type=$1 AND target_id=$2::uuid AND deleted_at IS NULL
	`, targetType, targetID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id::text, author_id::text, target_type, target_id::text, rating, comment, metadata, created_at::text, updated_at::text
		FROM entity_reviews
		WHERE target_type=$1 AND target_id=$2::uuid AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, targetType, targetID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]EntityReview, 0)
	for rows.Next() {
		item, err := scanEntityReview(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetRatingSummary(ctx context.Context, targetType, targetID string) (RatingSummary, error) {
	row := r.db.QueryRow(ctx, `
		SELECT target_type, target_id::text, rating_avg::float8, rating_count, updated_at::text
		FROM entity_rating_summaries
		WHERE target_type=$1 AND target_id=$2::uuid
	`, targetType, targetID)
	var item RatingSummary
	if err := row.Scan(&item.TargetType, &item.TargetID, &item.RatingAvg, &item.RatingCount, &item.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RatingSummary{TargetType: targetType, TargetID: targetID, RatingAvg: 5, RatingCount: 0}, nil
		}
		return RatingSummary{}, err
	}
	return item, nil
}

func (r *Repository) RecalculateClientRatingTx(ctx context.Context, tx pgx.Tx, clientID string) error {
	_, err := tx.Exec(ctx, `
		WITH stats AS (
			SELECT COALESCE(ROUND(AVG(rating)::numeric, 2), 5) AS rating_avg,
			       COUNT(*)::int AS rating_count
			FROM reviews
			WHERE reviewee_id=$1::uuid
			  AND reviewee_role='client'
			  AND deleted_at IS NULL
		)
		UPDATE client_profiles cp
		SET rating_avg=stats.rating_avg,
		    rating_count=stats.rating_count,
		    completed_orders=(
		    	SELECT COUNT(*)::int
		    	FROM orders
		    	WHERE client_id=$1::uuid AND status='completed' AND deleted_at IS NULL
		    ),
		    updated_at=NOW()
		FROM stats
		WHERE cp.user_id=$1::uuid
	`, clientID)
	return err
}

func (r *Repository) UpdateEntityReviewOwnedTx(ctx context.Context, tx pgx.Tx, id, userID string, rating int, comment *string) (EntityReview, error) {
	row := tx.QueryRow(ctx, `
		UPDATE entity_reviews
		SET rating=$3,
		    comment=$4,
		    updated_at=NOW()
		WHERE id=$1 AND author_id=$2 AND deleted_at IS NULL AND COALESCE(metadata->>'source','') <> 'course_review'
		RETURNING id::text, author_id::text, target_type, target_id::text, rating, comment, metadata, created_at::text, updated_at::text
	`, id, userID, rating, comment)
	return scanEntityReview(row)
}

func (r *Repository) DeleteEntityReviewOwnedTx(ctx context.Context, tx pgx.Tx, id, userID string) (EntityReview, error) {
	row := tx.QueryRow(ctx, `
		UPDATE entity_reviews
		SET deleted_at=NOW(),
		    updated_at=NOW()
		WHERE id=$1 AND author_id=$2 AND deleted_at IS NULL AND COALESCE(metadata->>'source','') <> 'course_review'
		RETURNING id::text, author_id::text, target_type, target_id::text, rating, comment, metadata, created_at::text, updated_at::text
	`, id, userID)
	return scanEntityReview(row)
}

func (r *Repository) UpdateCourseOwnerMirrorTx(ctx context.Context, tx pgx.Tx, courseReviewID string, rating int, comment *string) (targetID string, updated bool, err error) {
	row := tx.QueryRow(ctx, `
		UPDATE entity_reviews
		SET rating=$2,
		    comment=$3,
		    updated_at=NOW()
		WHERE metadata->>'course_review_id'=$1
		  AND target_type='user'
		  AND deleted_at IS NULL
		RETURNING target_id::text
	`, courseReviewID, rating, comment)
	if err := row.Scan(&targetID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return targetID, true, nil
}

func (r *Repository) DeleteCourseOwnerMirrorTx(ctx context.Context, tx pgx.Tx, courseReviewID string) (targetID string, deleted bool, err error) {
	row := tx.QueryRow(ctx, `
		UPDATE entity_reviews
		SET deleted_at=NOW(),
		    updated_at=NOW()
		WHERE metadata->>'course_review_id'=$1
		  AND target_type='user'
		  AND deleted_at IS NULL
		RETURNING target_id::text
	`, courseReviewID)
	if err := row.Scan(&targetID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return targetID, true, nil
}

func scanEntityReview(row interface{ Scan(dest ...any) error }) (EntityReview, error) {
	var item EntityReview
	var metadata []byte
	if err := row.Scan(&item.ID, &item.AuthorID, &item.TargetType, &item.TargetID, &item.Rating, &item.Comment, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return EntityReview{}, err
	}
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	item.Metadata = json.RawMessage(metadata)
	return item, nil
}
