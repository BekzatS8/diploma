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

func (r *Repository) CreateUserWithProfile(ctx context.Context, user User, profileName, phone string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, true)
	`, user.ID, strings.ToLower(user.Email), user.PasswordHash, user.Role)
	if err != nil {
		return mapPgError(err)
	}

	switch user.Role {
	case "client":
		_, err = tx.Exec(ctx, `
			INSERT INTO client_profiles (user_id, company_name, phone)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''))
		`, user.ID, profileName, phone)
	case "executor":
		_, err = tx.Exec(ctx, `
			INSERT INTO executor_profiles (user_id, display_name)
			VALUES ($1, NULLIF($2, ''))
		`, user.ID, profileName)
	case "coach":
		_, err = tx.Exec(ctx, `
			INSERT INTO coach_profiles (user_id, display_name)
			VALUES ($1, NULLIF($2, ''))
		`, user.ID, profileName)
	default:
		return ErrInvalidRole
	}
	if err != nil {
		return fmt.Errorf("insert profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *Repository) CreateAdmin(ctx context.Context, user User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, 'admin', true)
	`, user.ID, strings.ToLower(user.Email), user.PasswordHash)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, is_active, created_at
		FROM users
		WHERE email = $1
	`, strings.ToLower(email)).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
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
		SELECT id, email, password_hash, role, is_active, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
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
