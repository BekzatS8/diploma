package ratingsanctions

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db                          *pgxpool.Pool
	autoAssignCourseOnLowRating bool
	defaultLowRatingCourseID    string
}

type RepositoryOptions struct {
	AutoAssignCourseOnLowRating bool
	DefaultLowRatingCourseID    string
}

func NewRepository(db *pgxpool.Pool, opts RepositoryOptions) *Repository {
	return &Repository{db: db, autoAssignCourseOnLowRating: opts.AutoAssignCourseOnLowRating, defaultLowRatingCourseID: strings.TrimSpace(opts.DefaultLowRatingCourseID)}
}

type RatingInfo struct {
	ExecutorID         string  `json:"executor_id"`
	ReviewsCountTotal  int     `json:"reviews_count_total"`
	ReviewsCountRecent int     `json:"reviews_count_recent"`
	AvgRatingRecent    float64 `json:"avg_rating_recent"`
	AvgRatingTotal     float64 `json:"avg_rating_total"`
}

type Sanction struct {
	ID         string     `json:"id"`
	ExecutorID string     `json:"executor_id"`
	Status     string     `json:"status"`
	Reason     string     `json:"reason"`
	Severity   int        `json:"severity"`
	StartedAt  time.Time  `json:"started_at"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	ExpiredAt  *time.Time `json:"expired_at,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type ExpireResult struct {
	ExpiredCount int64 `json:"expired_count"`
}

type RecalculateResult struct {
	SanctionCreated        bool
	SanctionID             string
	SanctionReason         string
	AutoCourseAssigned     bool
	AutoCourseAssignmentID string
	AutoCourseID           string
}

func (r *Repository) HasActiveLowRating(ctx context.Context, executorID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM sanctions
			WHERE executor_id=$1 AND status='active' AND reason IN ('low_rating_first','low_rating_repeat')
			  AND (ends_at IS NULL OR ends_at > NOW())
		)
	`, executorID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetRating(ctx context.Context, executorID string) (RatingInfo, error) {
	var info RatingInfo
	info.ExecutorID = executorID
	if err := r.db.QueryRow(ctx, `SELECT rating_count, COALESCE(rating_avg::float8, 5) FROM executor_profiles WHERE user_id=$1`, executorID).Scan(&info.ReviewsCountTotal, &info.AvgRatingTotal); err != nil {
		return RatingInfo{}, err
	}
	if info.ReviewsCountTotal == 0 {
		info.AvgRatingTotal = 5
	}
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(AVG(rating)::float8, 5)
		FROM (
			SELECT rating
			FROM reviews
			WHERE reviewee_id=$1 AND reviewee_role='executor' AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 10
		) t
	`, executorID).Scan(&info.ReviewsCountRecent, &info.AvgRatingRecent); err != nil {
		return RatingInfo{}, err
	}
	if info.ReviewsCountRecent == 0 {
		info.AvgRatingRecent = 5
	}
	return info, nil
}

func expireDueTx(ctx context.Context, tx pgx.Tx, actorID, source, executorID string) (int64, error) {
	var executorArg any
	if executorID != "" {
		executorArg = executorID
	}
	res, err := tx.Exec(ctx, `
		UPDATE sanctions
		SET status='expired',
			expired_at=COALESCE(expired_at, NOW()),
			metadata = metadata || jsonb_build_object(
				'expired_by', NULLIF($1::text,''),
				'expired_source', $2::text,
				'expired_at', NOW()
			)
		WHERE status='active'
		  AND ends_at IS NOT NULL
		  AND ends_at <= NOW()
		  AND ($3::uuid IS NULL OR executor_id=$3::uuid)
	`, actorID, source, executorArg)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

func (r *Repository) RecalculateAndApply(ctx context.Context, tx pgx.Tx, executorID string) (RecalculateResult, error) {
	result := RecalculateResult{}
	var total int
	var avgTotal float64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*), COALESCE(AVG(rating)::float8,5) FROM reviews WHERE reviewee_id=$1 AND reviewee_role='executor' AND deleted_at IS NULL`, executorID).Scan(&total, &avgTotal); err != nil {
		return result, err
	}
	if total == 0 {
		avgTotal = 5
	}
	var recentCount int
	var avgRecent float64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*), COALESCE(AVG(rating)::float8,5) FROM (SELECT rating FROM reviews WHERE reviewee_id=$1 AND reviewee_role='executor' AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 10) t`, executorID).Scan(&recentCount, &avgRecent); err != nil {
		return result, err
	}
	if recentCount == 0 {
		avgRecent = 5
	}

	if _, err := tx.Exec(ctx, `UPDATE executor_profiles SET rating_avg=$2, rating_count=$3, updated_at=NOW() WHERE user_id=$1`, executorID, avgTotal, total); err != nil {
		return result, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO executor_rating_snapshots(executor_id, rating_avg, rating_count, snapshot_reason) VALUES($1,$2,$3,$4)`, executorID, avgRecent, recentCount, "review_created"); err != nil {
		return result, err
	}

	if _, err := expireDueTx(ctx, tx, "", "rating_recalculate", executorID); err != nil {
		return result, err
	}

	if recentCount == 0 || avgRecent >= 3.0 {
		return result, nil
	}

	var activeExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sanctions WHERE executor_id=$1 AND status='active' AND reason IN ('low_rating_first','low_rating_repeat') AND (ends_at IS NULL OR ends_at>NOW()))`, executorID).Scan(&activeExists); err != nil {
		return result, err
	}
	if activeExists {
		return result, nil
	}

	var hadBefore bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sanctions WHERE executor_id=$1 AND reason IN ('low_rating_first','low_rating_repeat'))`, executorID).Scan(&hadBefore); err != nil {
		return result, err
	}
	reason := "low_rating_first"
	severity := 2
	duration := "7 days"
	assignmentSource := "sanction_low_rating_first"
	if hadBefore {
		reason = "low_rating_repeat"
		severity = 4
		duration = "30 days"
		assignmentSource = "sanction_low_rating_repeat"
	}

	var sanctionID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO sanctions(executor_id, status, reason, severity, started_at, ends_at, metadata)
		VALUES($1,'active',$2,$3,NOW(),NOW()+($4::interval), jsonb_build_object(
			'trigger','rating_below_threshold',
			'window','last_10_reviews',
			'course_followup_intent','low_rating_followup',
			'auto_assign_enabled',$5,
			'default_course_id', NULLIF($6,'')
		))
		RETURNING id
	`, executorID, reason, severity, duration, r.autoAssignCourseOnLowRating, r.defaultLowRatingCourseID).Scan(&sanctionID); err != nil {
		return result, err
	}
	result.SanctionCreated = true
	result.SanctionID = sanctionID
	result.SanctionReason = reason

	autoCreated := false
	autoAssignmentID := ""
	if r.autoAssignCourseOnLowRating && r.defaultLowRatingCourseID != "" {
		var published bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM courses WHERE id=$1 AND deleted_at IS NULL AND status='published')`, r.defaultLowRatingCourseID).Scan(&published); err != nil {
			return result, err
		}
		if published {
			err := tx.QueryRow(ctx, `
				INSERT INTO course_assignments(course_id, executor_id, sanction_id, assigned_by, reason, source, status)
				VALUES($1,$2,$3,NULL,'Auto-assigned due to low rating sanction',$4,'assigned')
				ON CONFLICT (course_id, executor_id) WHERE status IN ('assigned','in_progress') DO NOTHING
				RETURNING id
			`, r.defaultLowRatingCourseID, executorID, sanctionID, assignmentSource).Scan(&autoAssignmentID)
			if err == nil {
				autoCreated = true
			} else if err != pgx.ErrNoRows {
				return result, err
			}
		}
	}

	_, err := tx.Exec(ctx, `
		UPDATE sanctions
		SET metadata = metadata || jsonb_build_object(
			'auto_assign_attempted', $2,
			'auto_assignment_created', $3,
			'assignment_source', $4
		)
		WHERE id=$1
	`, sanctionID, r.autoAssignCourseOnLowRating && r.defaultLowRatingCourseID != "", autoCreated, assignmentSource)
	if err != nil {
		return result, err
	}
	result.AutoCourseAssigned = autoCreated
	result.AutoCourseAssignmentID = autoAssignmentID
	result.AutoCourseID = r.defaultLowRatingCourseID
	return result, nil
}

func (r *Repository) ListMy(ctx context.Context, executorID string) ([]Sanction, error) {
	rows, err := r.db.Query(ctx, `SELECT id, executor_id, status, reason, severity, started_at, ends_at, expired_at, resolved_at FROM sanctions WHERE executor_id=$1 ORDER BY started_at DESC`, executorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Sanction, 0)
	for rows.Next() {
		var s Sanction
		if err := rows.Scan(&s.ID, &s.ExecutorID, &s.Status, &s.Reason, &s.Severity, &s.StartedAt, &s.EndsAt, &s.ExpiredAt, &s.ResolvedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) ListAdmin(ctx context.Context, limit, offset int) ([]Sanction, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM sanctions`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id, executor_id, status, reason, severity, started_at, ends_at, expired_at, resolved_at FROM sanctions ORDER BY started_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]Sanction, 0)
	for rows.Next() {
		var s Sanction
		if err := rows.Scan(&s.ID, &s.ExecutorID, &s.Status, &s.Reason, &s.Severity, &s.StartedAt, &s.EndsAt, &s.ExpiredAt, &s.ResolvedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *Repository) GetSanction(ctx context.Context, id string) (Sanction, error) {
	row := r.db.QueryRow(ctx, `SELECT id, executor_id, status, reason, severity, started_at, ends_at, expired_at, resolved_at FROM sanctions WHERE id=$1`, id)
	var s Sanction
	err := row.Scan(&s.ID, &s.ExecutorID, &s.Status, &s.Reason, &s.Severity, &s.StartedAt, &s.EndsAt, &s.ExpiredAt, &s.ResolvedAt)
	return s, err
}

func (r *Repository) Lift(ctx context.Context, id, actorID string) error {
	res, err := r.db.Exec(ctx, `
		UPDATE sanctions
		SET status='resolved',
			resolved_at=NOW(),
			metadata = metadata || jsonb_build_object('resolved_by',$2,'resolve_source','admin_lift')
		WHERE id=$1 AND status IN ('active','expired')
	`, id, actorID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ResolveExpired(ctx context.Context, id, actorID string) error {
	res, err := r.db.Exec(ctx, `
		UPDATE sanctions
		SET status='resolved',
			resolved_at=NOW(),
			metadata = metadata || jsonb_build_object('resolved_by',$2,'resolve_source','admin_resolve')
		WHERE id=$1 AND status='expired'
	`, id, actorID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ExpireDue(ctx context.Context, actorID, source string) (ExpireResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ExpireResult{}, err
	}
	defer tx.Rollback(ctx)
	count, err := expireDueTx(ctx, tx, actorID, source, "")
	if err != nil {
		return ExpireResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExpireResult{}, err
	}
	return ExpireResult{ExpiredCount: count}, nil
}
