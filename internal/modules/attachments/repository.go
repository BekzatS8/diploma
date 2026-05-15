package attachments

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, item Attachment) error {
	metadata, err := item.Metadata.jsonBytes()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO attachments(id, upload_id, target_type, target_id, sort_order, metadata, created_at)
		VALUES($1, $2, $3, $4, $5, $6::jsonb, $7)
	`, item.ID, item.UploadID, item.TargetType, item.TargetID, item.SortOrder, string(metadata), item.CreatedAt)
	return err
}

func (r *Repository) CountByTarget(ctx context.Context, targetType TargetType, targetID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM attachments WHERE target_type=$1 AND target_id=$2`, targetType, targetID).Scan(&count)
	return count, err
}

func (r *Repository) ListByTarget(ctx context.Context, targetType TargetType, targetID string) ([]Attachment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, upload_id, target_type, target_id, sort_order, metadata, created_at
		FROM attachments
		WHERE target_type=$1 AND target_id=$2
		ORDER BY sort_order ASC, created_at ASC
	`, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Attachment, 0)
	for rows.Next() {
		item, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (Attachment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, upload_id, target_type, target_id, sort_order, metadata, created_at
		FROM attachments
		WHERE id=$1
	`, id)
	return scanAttachment(row)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM attachments WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdateSortOrder(ctx context.Context, id string, sortOrder int) error {
	tag, err := r.db.Exec(ctx, `UPDATE attachments SET sort_order=$1 WHERE id=$2`, sortOrder, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanAttachment(row interface{ Scan(dest ...any) error }) (Attachment, error) {
	var item Attachment
	var metadata []byte
	err := row.Scan(&item.ID, &item.UploadID, &item.TargetType, &item.TargetID, &item.SortOrder, &metadata, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	if err != nil {
		return Attachment{}, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return Attachment{}, err
		}
	}
	if item.Metadata == nil {
		item.Metadata = Metadata{}
	}
	return item, nil
}
