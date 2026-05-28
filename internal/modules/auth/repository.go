package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (r *Repository) CreateUserWithProfile(ctx context.Context, user User, in RegisterInput) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, is_active, verification_status)
		VALUES ($1, $2, $3, $4, true, $5)
	`, user.ID, strings.ToLower(user.Email), user.PasswordHash, user.Role, defaultVerificationStatus(user.VerificationStatus))
	if err != nil {
		return mapPgError(err)
	}

	switch user.Role {
	case "client":
		_, err = tx.Exec(ctx, `
			INSERT INTO client_profiles (
				user_id, company_name, phone, client_type, tax_number, contact_name,
				contact_position, address, about, website
			)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''))
		`, user.ID, firstNonEmpty(in.ProfileName, in.ContactName), in.Phone, in.ClientType, in.TaxNumber, in.ContactName, in.ContactPosition, in.Address, in.About, in.Website)
	case "executor":
		_, err = tx.Exec(ctx, `
			INSERT INTO executor_profiles (
				user_id, display_name, first_name, last_name, middle_name, iin, phone, city,
				experience_level, education, work_format, hourly_rate, about, website, verification_status
			)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), $12, NULLIF($13, ''), NULLIF($14, ''), 'pending')
		`, user.ID, firstNonEmpty(in.ProfileName, strings.TrimSpace(in.FirstName+" "+in.LastName)), in.FirstName, in.LastName, in.MiddleName, in.IIN, in.Phone, in.City, in.ExperienceLevel, in.Education, in.WorkFormat, in.HourlyRate, in.About, in.Website)
	case "coach":
		expertise := strings.Join(in.Specializations, ", ")
		if expertise == "" {
			expertise = strings.TrimSpace(in.ExperienceLevel)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO coach_profiles (user_id, display_name, bio, expertise, website)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''))
		`, user.ID, in.ProfileName, in.About, expertise, in.Website)
	default:
		return ErrInvalidRole
	}
	if err != nil {
		return fmt.Errorf("insert profile: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets(user_id, balance, currency)
		VALUES($1, 0, 'KZT')
		ON CONFLICT (user_id) DO NOTHING
	`, user.ID); err != nil {
		return fmt.Errorf("insert wallet: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *Repository) CreateAdmin(ctx context.Context, user User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, is_active, verification_status)
		VALUES ($1, $2, $3, 'admin', true, 'verified')
	`, user.ID, strings.ToLower(user.Email), user.PasswordHash)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, is_active, verification_status,
		       verification_submitted_at, verification_reviewed_at, verification_reviewed_by::text,
		       verification_rejection_reason, created_at
		FROM users
		WHERE email = $1
	`, strings.ToLower(email)).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.VerificationStatus, &u.VerificationSubmittedAt, &u.VerificationReviewedAt, &u.VerificationReviewedBy, &u.VerificationRejectionReason, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	return u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, is_active, verification_status,
		       verification_submitted_at, verification_reviewed_at, verification_reviewed_by::text,
		       verification_rejection_reason, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.VerificationStatus, &u.VerificationSubmittedAt, &u.VerificationReviewedAt, &u.VerificationReviewedBy, &u.VerificationRejectionReason, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUnauthorized
		}
		return User{}, err
	}
	return u, nil
}

func (r *Repository) CreateRefreshToken(ctx context.Context, token RefreshToken) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetRefreshToken(ctx context.Context, tokenID string) (RefreshToken, error) {
	var t RefreshToken
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE id = $1
	`, tokenID).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshToken{}, ErrUnauthorized
		}
		return RefreshToken{}, err
	}
	if t.RevokedAt != nil || t.ExpiresAt.Before(time.Now()) {
		return RefreshToken{}, ErrUnauthorized
	}
	return t, nil
}

func (r *Repository) RotateRefreshToken(ctx context.Context, oldTokenID string, newToken RefreshToken) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, oldTokenID, now)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, newToken.ID, newToken.UserID, newToken.TokenHash, newToken.ExpiresAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenID string, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, tokenID, userID)
	return err
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrEmailAlreadyExists
	}
	return err
}

func defaultVerificationStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "verified"
	}
	return status
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
