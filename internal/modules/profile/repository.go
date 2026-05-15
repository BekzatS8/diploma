package profile

import (
	"context"
	"encoding/json"
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
		var companyName, taxNumber, phone, about, clientType, contactName, contactPosition, address *string
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `
			SELECT company_name, tax_number, phone, about, client_type, contact_name, contact_position, address, avatar_upload_id::text
			FROM client_profiles WHERE user_id=$1
		`, userID).Scan(&companyName, &taxNumber, &phone, &about, &clientType, &contactName, &contactPosition, &address, &avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return map[string]any{}, nil
			}
			return nil, err
		}
		return map[string]any{"company_name": companyName, "tax_number": taxNumber, "phone": phone, "about": about, "client_type": clientType, "contact_name": contactName, "contact_position": contactPosition, "address": address, "avatar_upload_id": avatarUploadID}, nil
	case "executor":
		var displayName, bio, firstName, lastName, middleName, iin, phone, city, experienceLevel, education, workFormat, about, verificationStatus, rejectionReason *string
		var years int
		var hourlyRate *float64
		var verifiedAt, rejectedAt *string
		var specializations []byte
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `
			SELECT display_name, bio, years_experience,
			       first_name, last_name, middle_name, iin, phone, city, experience_level,
			       specializations, education, work_format, hourly_rate, about,
			       verification_status, verified_at::text, rejected_at::text, rejection_reason,
			       avatar_upload_id::text
			FROM executor_profiles WHERE user_id=$1
		`, userID).Scan(&displayName, &bio, &years,
			&firstName, &lastName, &middleName, &iin, &phone, &city, &experienceLevel,
			&specializations, &education, &workFormat, &hourlyRate, &about,
			&verificationStatus, &verifiedAt, &rejectedAt, &rejectionReason,
			&avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return map[string]any{}, nil
			}
			return nil, err
		}
		var specs []string
		if len(specializations) > 0 {
			_ = json.Unmarshal(specializations, &specs)
		}
		if specs == nil {
			specs = []string{}
		}
		return map[string]any{
			"profile_name": displayName, "bio": bio, "years_experience": years,
			"first_name": firstName, "last_name": lastName, "middle_name": middleName,
			"iin": iin, "phone": phone, "city": city, "experience_level": experienceLevel,
			"specializations": specs, "education": education, "work_format": workFormat,
			"hourly_rate": hourlyRate, "about": about, "verification_status": verificationStatus,
			"verified_at": verifiedAt, "rejected_at": rejectedAt, "rejection_reason": rejectionReason,
			"avatar_upload_id": avatarUploadID,
		}, nil
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
				client_type = COALESCE($5, client_type),
				tax_number = COALESCE($6, tax_number),
				contact_name = COALESCE($7, contact_name),
				contact_position = COALESCE($8, contact_position),
				address = COALESCE($9, address),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.CompanyName, req.Phone, req.About, req.ClientType, req.TaxNumber, req.ContactName, req.ContactPosition, req.Address)
		return err
	case "executor":
		var specializationsJSON *string
		if req.Specializations != nil {
			raw, err := json.Marshal(req.Specializations)
			if err != nil {
				return err
			}
			value := string(raw)
			specializationsJSON = &value
		}
		_, err := r.db.Exec(ctx, `
			UPDATE executor_profiles
			SET display_name = COALESCE($2, display_name),
				bio = COALESCE($3, bio),
				years_experience = COALESCE($4, years_experience),
				first_name = COALESCE($5, first_name),
				last_name = COALESCE($6, last_name),
				middle_name = COALESCE($7, middle_name),
				iin = COALESCE($8, iin),
				phone = COALESCE($9, phone),
				city = COALESCE($10, city),
				experience_level = COALESCE($11, experience_level),
				specializations = COALESCE($12::jsonb, specializations),
				education = COALESCE($13, education),
				work_format = COALESCE($14, work_format),
				hourly_rate = COALESCE($15, hourly_rate),
				about = COALESCE($16, about),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.ProfileName, req.Bio, req.YearsExperience,
			req.FirstName, req.LastName, req.MiddleName, req.IIN, req.Phone, req.City,
			req.ExperienceLevel, specializationsJSON, req.Education, req.WorkFormat, req.HourlyRate, req.About)
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
