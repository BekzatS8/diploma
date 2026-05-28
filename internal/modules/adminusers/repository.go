package adminusers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("user not found")
	ErrNotExecutor  = errors.New("user is not an executor")
	ErrAlreadyCoach = errors.New("user is already a coach")
	ErrNotCoach     = errors.New("user does not have coach capabilities")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListExecutors(ctx context.Context, q string, page, pageSize int) ([]ExecutorListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	search := "%" + strings.TrimSpace(q) + "%"

	var total int64
	countQuery := `
		SELECT COUNT(*)
		FROM users u
		INNER JOIN executor_profiles ep ON ep.user_id = u.id
		WHERE u.role = 'executor'
		  AND ($1 = '' OR u.email ILIKE $2 OR COALESCE(ep.display_name, '') ILIKE $2)
	`
	if err := r.db.QueryRow(ctx, countQuery, strings.TrimSpace(q), search).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, u.email, COALESCE(ep.display_name, ''), u.is_active,
		       EXISTS(SELECT 1 FROM coach_profiles cp WHERE cp.user_id = u.id),
		       COALESCE(ep.rating_avg::float8, 0), ep.rating_count, u.created_at
		FROM users u
		INNER JOIN executor_profiles ep ON ep.user_id = u.id
		WHERE u.role = 'executor'
		  AND ($1 = '' OR u.email ILIKE $2 OR COALESCE(ep.display_name, '') ILIKE $2)
		ORDER BY ep.rating_count DESC, ep.rating_avg DESC, u.created_at DESC
		LIMIT $3 OFFSET $4
	`, strings.TrimSpace(q), search, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ExecutorListItem, 0, pageSize)
	for rows.Next() {
		var item ExecutorListItem
		var createdAt time.Time
		if err := rows.Scan(
			&item.UserID, &item.Email, &item.DisplayName, &item.IsActive, &item.IsCoach,
			&item.RatingAvg, &item.RatingCount, &createdAt,
		); err != nil {
			return nil, 0, err
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) PromoteExecutorToCoach(ctx context.Context, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var role string
	var isActive bool
	err = tx.QueryRow(ctx, `
		SELECT role, is_active FROM users WHERE id = $1 FOR UPDATE
	`, userID).Scan(&role, &isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if role != "executor" {
		if role == "admin" {
			return ErrNotExecutor
		}
		return ErrNotExecutor
	}
	if !isActive {
		return fmt.Errorf("cannot promote inactive user")
	}

	var hasCoach bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM coach_profiles WHERE user_id = $1)`, userID).Scan(&hasCoach); err != nil {
		return err
	}
	if hasCoach {
		return ErrAlreadyCoach
	}

	var displayName, bio, about, website *string
	var specializations []byte
	var avatarUploadID *string
	err = tx.QueryRow(ctx, `
		SELECT display_name, bio, about, specializations, website, avatar_upload_id::text
		FROM executor_profiles WHERE user_id = $1
	`, userID).Scan(&displayName, &bio, &about, &specializations, &website, &avatarUploadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotExecutor
		}
		return err
	}

	expertise := specializationsToExpertise(specializations)
	coachBio := firstNonEmpty(ptrStr(about), ptrStr(bio))

	var avatarArg any
	if avatarUploadID != nil && strings.TrimSpace(*avatarUploadID) != "" {
		avatarArg = strings.TrimSpace(*avatarUploadID)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO coach_profiles (user_id, display_name, bio, expertise, website, avatar_upload_id)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6::uuid)
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = COALESCE(EXCLUDED.display_name, coach_profiles.display_name),
			bio = COALESCE(EXCLUDED.bio, coach_profiles.bio),
			expertise = COALESCE(EXCLUDED.expertise, coach_profiles.expertise),
			website = COALESCE(EXCLUDED.website, coach_profiles.website),
			avatar_upload_id = COALESCE(EXCLUDED.avatar_upload_id, coach_profiles.avatar_upload_id),
			updated_at = NOW()
	`, userID, ptrStr(displayName), coachBio, expertise, ptrStr(website), avatarArg)
	if err != nil {
		return fmt.Errorf("upsert coach profile: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) RevokeCoachFromExecutor(ctx context.Context, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var role string
	err = tx.QueryRow(ctx, `SELECT role FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if role != "executor" {
		return ErrNotExecutor
	}

	var hasCoach bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM coach_profiles WHERE user_id = $1)`, userID).Scan(&hasCoach); err != nil {
		return err
	}
	if !hasCoach {
		return ErrNotCoach
	}

	tag, err := tx.Exec(ctx, `DELETE FROM coach_profiles WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotCoach
	}

	return tx.Commit(ctx)
}

func specializationsToExpertise(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var specs []string
	if err := json.Unmarshal(raw, &specs); err != nil {
		return ""
	}
	return strings.Join(specs, ", ")
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
