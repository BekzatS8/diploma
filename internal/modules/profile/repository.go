package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByRole(ctx context.Context, userID, role string) (map[string]any, error) {
	switch role {
	case "client":
		var companyName, phone, about *string
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `SELECT company_name, phone, about, avatar_upload_id::text FROM client_profiles WHERE user_id=$1`, userID).Scan(&companyName, &phone, &about, &avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return map[string]any{}, nil
			}
			return nil, err
		}
		return map[string]any{"company_name": companyName, "phone": phone, "about": about, "avatar_upload_id": avatarUploadID}, nil
	case "executor":
		var displayName, bio *string
		var years int
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `SELECT display_name, bio, years_experience, avatar_upload_id::text FROM executor_profiles WHERE user_id=$1`, userID).Scan(&displayName, &bio, &years, &avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return map[string]any{}, nil
			}
			return nil, err
		}
		return map[string]any{"profile_name": displayName, "bio": bio, "years_experience": years, "avatar_upload_id": avatarUploadID}, nil
	case "coach":
		var displayName, bio, expertise *string
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `SELECT display_name, bio, expertise, avatar_upload_id::text FROM coach_profiles WHERE user_id=$1`, userID).Scan(&displayName, &bio, &expertise, &avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return map[string]any{}, nil
			}
			return nil, err
		}
		return map[string]any{"profile_name": displayName, "bio": bio, "expertise": expertise, "avatar_upload_id": avatarUploadID}, nil
	default:
		return nil, fmt.Errorf("unsupported role")
	}
}

func (r *Repository) SetAvatarUploadID(ctx context.Context, userID, role, uploadID string) error {
	var err error
	switch role {
	case "client":
		_, err = r.db.Exec(ctx, `UPDATE client_profiles SET avatar_upload_id=$2, updated_at=NOW() WHERE user_id=$1`, userID, uploadID)
	case "executor":
		_, err = r.db.Exec(ctx, `UPDATE executor_profiles SET avatar_upload_id=$2, updated_at=NOW() WHERE user_id=$1`, userID, uploadID)
	case "coach":
		_, err = r.db.Exec(ctx, `UPDATE coach_profiles SET avatar_upload_id=$2, updated_at=NOW() WHERE user_id=$1`, userID, uploadID)
	default:
		err = fmt.Errorf("unsupported role")
	}
	return err
}

func (r *Repository) ClearAvatarUploadID(ctx context.Context, userID, role string) error {
	var err error
	switch role {
	case "client":
		_, err = r.db.Exec(ctx, `UPDATE client_profiles SET avatar_upload_id=NULL, updated_at=NOW() WHERE user_id=$1`, userID)
	case "executor":
		_, err = r.db.Exec(ctx, `UPDATE executor_profiles SET avatar_upload_id=NULL, updated_at=NOW() WHERE user_id=$1`, userID)
	case "coach":
		_, err = r.db.Exec(ctx, `UPDATE coach_profiles SET avatar_upload_id=NULL, updated_at=NOW() WHERE user_id=$1`, userID)
	default:
		err = fmt.Errorf("unsupported role")
	}
	return err
}

func (r *Repository) PatchByRole(ctx context.Context, userID, role string, req UpdateProfileRequest) error {
	switch role {
	case "client":
		_, err := r.db.Exec(ctx, `
			UPDATE client_profiles
			SET company_name = COALESCE($2, company_name),
				phone = COALESCE($3, phone),
				about = COALESCE($4, about),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.CompanyName, req.Phone, req.About)
		return err
	case "executor":
		_, err := r.db.Exec(ctx, `
			UPDATE executor_profiles
			SET display_name = COALESCE($2, display_name),
				bio = COALESCE($3, bio),
				years_experience = COALESCE($4, years_experience),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.ProfileName, req.Bio, req.YearsExperience)
		return err
	case "coach":
		_, err := r.db.Exec(ctx, `
			UPDATE coach_profiles
			SET display_name = COALESCE($2, display_name),
				bio = COALESCE($3, bio),
				expertise = COALESCE($4, expertise),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.ProfileName, req.Bio, req.Expertise)
		return err
	default:
		return fmt.Errorf("unsupported role")
	}
}
