package orderreports

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, orderID, reporterID, reason string) (string, error) {
	id := uuid.NewString()
	_, err := r.db.Exec(ctx, `
		INSERT INTO order_reports (id, order_id, reporter_id, reason, status)
		VALUES ($1, $2, $3, $4, 'pending')
	`, id, orderID, reporterID, strings.TrimSpace(reason))
	if err != nil {
		if isUniqueViolation(err) {
			return "", ErrDuplicatePending
		}
		return "", err
	}
	return id, nil
}

func (r *Repository) OrderReportable(ctx context.Context, orderID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM orders
			WHERE id = $1 AND deleted_at IS NULL AND status = 'published'
		)
	`, orderID).Scan(&ok)
	return ok, err
}

func (r *Repository) List(ctx context.Context, status Status, page, pageSize int) ([]Report, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	where := ""
	args := []any{}
	if status != "" {
		where = "WHERE r.status = $1"
		args = append(args, status)
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM order_reports r `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	argPos := len(args) + 1
	query := fmt.Sprintf(`
		SELECT r.id::text, r.order_id::text, r.reporter_id::text, r.reason, r.status,
		       r.admin_id::text, r.admin_notes, r.created_at, r.reviewed_at,
		       o.title, o.description, o.budget_amount, o.currency, o.status,
		       u.email, COALESCE(ep.display_name, '')
		FROM order_reports r
		INNER JOIN orders o ON o.id = r.order_id
		INNER JOIN users u ON u.id = r.reporter_id
		LEFT JOIN executor_profiles ep ON ep.user_id = r.reporter_id
		%s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argPos, argPos+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Report, 0)
	for rows.Next() {
		var item Report
		var st string
		var reviewedAt *time.Time
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ReporterID, &item.Reason, &st,
			&item.AdminID, &item.AdminNotes, &item.CreatedAt, &reviewedAt,
			&item.OrderTitle, &item.OrderDesc, &item.OrderBudget, &item.OrderCurrency, &item.OrderStatus,
			&item.ReporterEmail, &item.ReporterName,
		); err != nil {
			return nil, 0, err
		}
		item.Status = Status(st)
		item.ReviewedAt = reviewedAt
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (Report, error) {
	row := r.db.QueryRow(ctx, `
		SELECT r.id::text, r.order_id::text, r.reporter_id::text, r.reason, r.status,
		       r.admin_id::text, r.admin_notes, r.created_at, r.reviewed_at,
		       o.title, o.description, o.budget_amount, o.currency, o.status,
		       u.email, COALESCE(ep.display_name, '')
		FROM order_reports r
		INNER JOIN orders o ON o.id = r.order_id
		INNER JOIN users u ON u.id = r.reporter_id
		LEFT JOIN executor_profiles ep ON ep.user_id = r.reporter_id
		WHERE r.id = $1
	`, id)
	var item Report
	var st string
	var reviewedAt *time.Time
	err := row.Scan(
		&item.ID, &item.OrderID, &item.ReporterID, &item.Reason, &st,
		&item.AdminID, &item.AdminNotes, &item.CreatedAt, &reviewedAt,
		&item.OrderTitle, &item.OrderDesc, &item.OrderBudget, &item.OrderCurrency, &item.OrderStatus,
		&item.ReporterEmail, &item.ReporterName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Report{}, ErrNotFound
		}
		return Report{}, err
	}
	item.Status = Status(st)
	item.ReviewedAt = reviewedAt
	return item, nil
}

func (r *Repository) Review(ctx context.Context, id, adminID string, newStatus Status, notes string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE order_reports
		SET status = $2,
		    admin_id = $3,
		    admin_notes = NULLIF($4, ''),
		    reviewed_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, id, newStatus, adminID, strings.TrimSpace(notes))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		item, err := r.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if item.Status != StatusPending {
			return ErrAlreadyReviewed
		}
		return ErrNotFound
	}
	return nil
}

func (r *Repository) AdminSoftDeleteOrder(ctx context.Context, orderID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE orders
		SET deleted_at = NOW(),
		    updated_at = NOW(),
		    cancelled_at = COALESCE(cancelled_at, NOW())
		WHERE id = $1 AND deleted_at IS NULL
	`, orderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
