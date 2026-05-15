package uploads

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, item Upload) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO uploads(id, author_id, file_path, original_name, mime_type, size_bytes, created_at)
		VALUES($1, $2, $3, $4, $5, $6, $7)
	`, item.ID, item.AuthorID, item.FilePath, item.OriginalName, item.MimeType, item.SizeBytes, item.CreatedAt)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (Upload, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, author_id, file_path, original_name, mime_type, size_bytes, created_at
		FROM uploads
		WHERE id=$1
	`, id)
	return scanUpload(row)
}

func (r *Repository) ListByAuthor(ctx context.Context, authorID string) ([]Upload, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, author_id, file_path, original_name, mime_type, size_bytes, created_at
		FROM uploads
		WHERE author_id=$1
		ORDER BY created_at DESC
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Upload, 0)
	for rows.Next() {
		item, err := scanUpload(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM uploads WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanUpload(row interface{ Scan(dest ...any) error }) (Upload, error) {
	var item Upload
	err := row.Scan(&item.ID, &item.AuthorID, &item.FilePath, &item.OriginalName, &item.MimeType, &item.SizeBytes, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Upload{}, ErrNotFound
	}
	return item, err
}
