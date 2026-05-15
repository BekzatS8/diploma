package leads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UserEmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email)=LOWER($1))`, email).Scan(&exists)
	return exists, err
}

func (r *Repository) Create(ctx context.Context, lead ExecutorLead, docs []ExecutorLeadDocument) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	specs, err := json.Marshal(lead.Specializations)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO executor_leads (
			id, email, password_hash, first_name, last_name, middle_name, iin, phone, city,
			experience_level, specializations, education, work_format, hourly_rate, about,
			terms_accepted, status, source, utm_source, utm_medium, utm_campaign, ip_address,
			user_agent, submitted_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11::jsonb, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22,
			$23, $24, $25, $26
		)
	`, lead.ID, lead.Email, lead.PasswordHash, lead.FirstName, lead.LastName, lead.MiddleName, lead.IIN, lead.Phone, lead.City,
		lead.ExperienceLevel, string(specs), lead.Education, lead.WorkFormat, lead.HourlyRate, lead.About,
		lead.TermsAccepted, lead.Status, lead.Source, lead.UTMSource, lead.UTMMedium, lead.UTMCampaign, lead.IPAddress,
		lead.UserAgent, lead.SubmittedAt, lead.CreatedAt, lead.UpdatedAt)
	if err != nil {
		return mapPgError(err)
	}

	for _, doc := range docs {
		_, err = tx.Exec(ctx, `
			INSERT INTO executor_lead_documents(id, lead_id, document_type, file_path, original_name, mime_type, size_bytes, created_at)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8)
		`, doc.ID, doc.LeadID, doc.DocumentType, doc.FilePath, doc.OriginalName, doc.MimeType, doc.SizeBytes, doc.CreatedAt)
		if err != nil {
			return mapPgError(err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) List(ctx context.Context, status Status, page, pageSize int) ([]ExecutorLead, int64, error) {
	where := ""
	args := []any{}
	if status != "" {
		where = "WHERE status=$1"
		args = append(args, status)
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM executor_leads "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	argPos := len(args) + 1
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, first_name, last_name, middle_name, iin, phone, city,
		       experience_level, specializations, education, work_format, hourly_rate, about,
		       terms_accepted, status, priority, notes, rejection_reason, source, utm_source,
		       utm_medium, utm_campaign, ip_address, user_agent, submitted_at, reviewed_at,
		       reviewed_by::text, converted_at, converted_user_id::text, created_at, updated_at
		FROM executor_leads
		%s
		ORDER BY priority DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argPos, argPos+1)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ExecutorLead, 0)
	for rows.Next() {
		item, err := scanLead(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (ExecutorLead, error) {
	item, err := r.getByID(ctx, id)
	if err != nil {
		return ExecutorLead{}, err
	}
	docs, err := r.ListDocuments(ctx, id)
	if err != nil {
		return ExecutorLead{}, err
	}
	item.Documents = docs
	return item, nil
}

func (r *Repository) ListDocuments(ctx context.Context, leadID string) ([]ExecutorLeadDocument, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, lead_id, document_type, file_path, original_name, mime_type, size_bytes, created_at
		FROM executor_lead_documents
		WHERE lead_id=$1
		ORDER BY created_at ASC
	`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	docs := make([]ExecutorLeadDocument, 0)
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status Status, adminID, notes string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE executor_leads
		SET status=$2, notes=COALESCE(NULLIF($4,''), notes), reviewed_at=NOW(), reviewed_by=$3, updated_at=NOW()
		WHERE id=$1 AND status <> 'converted'
	`, id, status, adminID, notes)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Reject(ctx context.Context, id, adminID, reason string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE executor_leads
		SET status='rejected', rejection_reason=$2, reviewed_at=NOW(), reviewed_by=$3, updated_at=NOW()
		WHERE id=$1 AND status <> 'converted'
	`, id, reason, adminID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ApproveAndConvert(ctx context.Context, id, adminID, notes string) (string, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	lead, err := r.getByIDTx(ctx, tx, id, true)
	if err != nil {
		return "", err
	}
	if lead.Status == StatusConverted {
		return "", ErrAlreadyConverted
	}
	if lead.Status == StatusRejected {
		return "", ErrInvalidStatus
	}

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email)=LOWER($1))`, lead.Email).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return "", ErrEmailExists
	}

	docs, err := r.listDocumentsTx(ctx, tx, id)
	if err != nil {
		return "", err
	}
	if !hasDocument(docs, DocumentIdentity) || !hasDocument(docs, DocumentEducation) {
		return "", ErrDocumentRequired
	}

	userID := uuid.NewString()
	now := time.Now()
	_, err = tx.Exec(ctx, `
		INSERT INTO users (
			id, email, password_hash, role, is_active, verification_status,
			verification_submitted_at, verification_reviewed_at, verification_reviewed_by
		)
		VALUES($1, $2, $3, 'executor', TRUE, 'verified', $4, $5, $6)
	`, userID, strings.ToLower(lead.Email), lead.PasswordHash, lead.SubmittedAt, now, adminID)
	if err != nil {
		return "", mapPgError(err)
	}

	specs, err := json.Marshal(lead.Specializations)
	if err != nil {
		return "", err
	}
	displayName := strings.TrimSpace(lead.FirstName + " " + lead.LastName)
	years := yearsFromExperienceLevel(lead.ExperienceLevel)
	_, err = tx.Exec(ctx, `
		INSERT INTO executor_profiles (
			user_id, display_name, bio, years_experience,
			first_name, last_name, middle_name, iin, phone, city, experience_level,
			specializations, education, work_format, hourly_rate, about,
			verification_status, verified_at
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10, $11,
			$12::jsonb, $13, $14, $15, $16,
			'verified', $17
		)
	`, userID, displayName, lead.About, years,
		lead.FirstName, lead.LastName, lead.MiddleName, lead.IIN, lead.Phone, lead.City, lead.ExperienceLevel,
		string(specs), lead.Education, lead.WorkFormat, lead.HourlyRate, lead.About, now)
	if err != nil {
		return "", err
	}

	for i, doc := range docs {
		uploadID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO uploads(id, author_id, file_path, original_name, mime_type, size_bytes, created_at)
			VALUES($1, $2, $3, $4, $5, $6, $7)
		`, uploadID, userID, doc.FilePath, doc.OriginalName, doc.MimeType, doc.SizeBytes, doc.CreatedAt)
		if err != nil {
			return "", mapPgError(err)
		}
		metadata, _ := json.Marshal(map[string]any{
			"document_type": doc.DocumentType,
			"lead_id":       lead.ID,
		})
		_, err = tx.Exec(ctx, `
			INSERT INTO attachments(id, upload_id, target_type, target_id, sort_order, metadata, created_at)
			VALUES($1, $2, 'profile_document', $3, $4, $5::jsonb, $6)
		`, uuid.NewString(), uploadID, userID, i, string(metadata), now)
		if err != nil {
			return "", mapPgError(err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE executor_leads
		SET status='converted', notes=COALESCE(NULLIF($3,''), notes), reviewed_at=$4,
		    reviewed_by=$2, converted_at=$4, converted_user_id=$5, updated_at=$4
		WHERE id=$1
	`, id, adminID, notes, now, userID)
	if err != nil {
		return "", err
	}

	return userID, tx.Commit(ctx)
}

func (r *Repository) getByID(ctx context.Context, id string) (ExecutorLead, error) {
	return r.getByIDTx(ctx, r.db, id, false)
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repository) getByIDTx(ctx context.Context, q queryer, id string, lock bool) (ExecutorLead, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	row := q.QueryRow(ctx, `
		SELECT id, email, password_hash, first_name, last_name, middle_name, iin, phone, city,
		       experience_level, specializations, education, work_format, hourly_rate, about,
		       terms_accepted, status, priority, notes, rejection_reason, source, utm_source,
		       utm_medium, utm_campaign, ip_address, user_agent, submitted_at, reviewed_at,
		       reviewed_by::text, converted_at, converted_user_id::text, created_at, updated_at
		FROM executor_leads
		WHERE id=$1
	`+suffix, id)
	return scanLead(row)
}

func (r *Repository) listDocumentsTx(ctx context.Context, tx pgx.Tx, leadID string) ([]ExecutorLeadDocument, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, lead_id, document_type, file_path, original_name, mime_type, size_bytes, created_at
		FROM executor_lead_documents
		WHERE lead_id=$1
		ORDER BY created_at ASC
	`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	docs := make([]ExecutorLeadDocument, 0)
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func scanLead(row interface{ Scan(dest ...any) error }) (ExecutorLead, error) {
	var item ExecutorLead
	var specs []byte
	err := row.Scan(
		&item.ID, &item.Email, &item.PasswordHash, &item.FirstName, &item.LastName, &item.MiddleName,
		&item.IIN, &item.Phone, &item.City, &item.ExperienceLevel, &specs, &item.Education,
		&item.WorkFormat, &item.HourlyRate, &item.About, &item.TermsAccepted, &item.Status,
		&item.Priority, &item.Notes, &item.RejectionReason, &item.Source, &item.UTMSource,
		&item.UTMMedium, &item.UTMCampaign, &item.IPAddress, &item.UserAgent, &item.SubmittedAt,
		&item.ReviewedAt, &item.ReviewedBy, &item.ConvertedAt, &item.ConvertedUserID,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExecutorLead{}, ErrNotFound
	}
	if err != nil {
		return ExecutorLead{}, err
	}
	if len(specs) > 0 {
		_ = json.Unmarshal(specs, &item.Specializations)
	}
	if item.Specializations == nil {
		item.Specializations = []string{}
	}
	return item, nil
}

func scanDocument(row interface{ Scan(dest ...any) error }) (ExecutorLeadDocument, error) {
	var doc ExecutorLeadDocument
	err := row.Scan(&doc.ID, &doc.LeadID, &doc.DocumentType, &doc.FilePath, &doc.OriginalName, &doc.MimeType, &doc.SizeBytes, &doc.CreatedAt)
	return doc, err
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "users_email") {
				return ErrEmailExists
			}
			return ErrDuplicate
		}
	}
	return err
}

func hasDocument(docs []ExecutorLeadDocument, documentType DocumentType) bool {
	for _, doc := range docs {
		if doc.DocumentType == documentType {
			return true
		}
	}
	return false
}

func yearsFromExperienceLevel(level string) int {
	switch strings.TrimSpace(strings.ToLower(level)) {
	case "0-1", "less_than_1", "до 1 года":
		return 0
	case "1-3", "1-2":
		return 1
	case "3-5":
		return 3
	case "5+", "5-plus", "more_than_5":
		return 5
	default:
		return 0
	}
}
