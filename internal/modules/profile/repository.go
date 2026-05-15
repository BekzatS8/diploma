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
	base := map[string]any{}
	var email string
	var createdAt string
	if err := r.db.QueryRow(ctx, `SELECT email, created_at::text FROM users WHERE id=$1`, userID).Scan(&email, &createdAt); err == nil {
		base["email"] = email
		base["platform_joined_at"] = createdAt
	}

	switch role {
	case "client":
		var companyName, taxNumber, phone, about, clientType, contactName, contactPosition, address, website *string
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `
			SELECT company_name, tax_number, phone, about, client_type, contact_name, contact_position, address, website, avatar_upload_id::text
			FROM client_profiles WHERE user_id=$1
		`, userID).Scan(&companyName, &taxNumber, &phone, &about, &clientType, &contactName, &contactPosition, &address, &website, &avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return base, nil
			}
			return nil, err
		}
		base["company_name"] = companyName
		base["tax_number"] = taxNumber
		base["phone"] = phone
		base["about"] = about
		base["client_type"] = clientType
		base["contact_name"] = contactName
		base["contact_position"] = contactPosition
		base["address"] = address
		base["website"] = website
		base["avatar_upload_id"] = avatarUploadID
		return base, nil
	case "executor":
		var displayName, bio, firstName, lastName, middleName, iin, phone, city, experienceLevel, education, workFormat, about, verificationStatus, rejectionReason, website *string
		var years, profileViews, responseRate int
		var hourlyRate *float64
		var verifiedAt, rejectedAt *string
		var specializations []byte
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `
			SELECT display_name, bio, years_experience,
			       first_name, last_name, middle_name, iin, phone, city, experience_level,
			       specializations, education, work_format, hourly_rate, about,
			       verification_status, verified_at::text, rejected_at::text, rejection_reason,
			       website, profile_views, response_rate, avatar_upload_id::text
			FROM executor_profiles WHERE user_id=$1
		`, userID).Scan(&displayName, &bio, &years,
			&firstName, &lastName, &middleName, &iin, &phone, &city, &experienceLevel,
			&specializations, &education, &workFormat, &hourlyRate, &about,
			&verificationStatus, &verifiedAt, &rejectedAt, &rejectionReason,
			&website, &profileViews, &responseRate, &avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return base, nil
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
		base["profile_name"] = displayName
		base["bio"] = bio
		base["years_experience"] = years
		base["first_name"] = firstName
		base["last_name"] = lastName
		base["middle_name"] = middleName
		base["iin"] = iin
		base["phone"] = phone
		base["city"] = city
		base["experience_level"] = experienceLevel
		base["specializations"] = specs
		base["education"] = education
		base["work_format"] = workFormat
		base["hourly_rate"] = hourlyRate
		base["about"] = about
		base["verification_status"] = verificationStatus
		base["verified_at"] = verifiedAt
		base["rejected_at"] = rejectedAt
		base["rejection_reason"] = rejectionReason
		base["website"] = website
		base["profile_views"] = profileViews
		base["response_rate"] = responseRate
		base["avatar_upload_id"] = avatarUploadID
		return base, nil
	case "coach":
		var displayName, bio, expertise, website *string
		var avatarUploadID *string
		err := r.db.QueryRow(ctx, `SELECT display_name, bio, expertise, website, avatar_upload_id::text FROM coach_profiles WHERE user_id=$1`, userID).Scan(&displayName, &bio, &expertise, &website, &avatarUploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return base, nil
			}
			return nil, err
		}
		base["profile_name"] = displayName
		base["bio"] = bio
		base["expertise"] = expertise
		base["website"] = website
		base["avatar_upload_id"] = avatarUploadID
		return base, nil
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
				website = COALESCE($10, website),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.CompanyName, req.Phone, req.About, req.ClientType, req.TaxNumber, req.ContactName, req.ContactPosition, req.Address, req.Website)
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
				website = COALESCE($17, website),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.ProfileName, req.Bio, req.YearsExperience,
			req.FirstName, req.LastName, req.MiddleName, req.IIN, req.Phone, req.City,
			req.ExperienceLevel, specializationsJSON, req.Education, req.WorkFormat, req.HourlyRate, req.About, req.Website)
		return err
	case "coach":
		_, err := r.db.Exec(ctx, `
			UPDATE coach_profiles
			SET display_name = COALESCE($2, display_name),
				bio = COALESCE($3, bio),
				expertise = COALESCE($4, expertise),
				website = COALESCE($5, website),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, req.ProfileName, req.Bio, req.Expertise, req.Website)
		return err
	default:
		return fmt.Errorf("unsupported role")
	}
}

func (r *Repository) ListProfileDocuments(ctx context.Context, userID string) ([]ProfileDocument, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, a.upload_id::text, u.file_path, u.original_name, u.mime_type, u.size_bytes, a.metadata, a.created_at::text
		FROM attachments a
		JOIN uploads u ON u.id = a.upload_id
		WHERE a.target_type='profile_document' AND a.target_id=$1::uuid
		ORDER BY a.sort_order ASC, a.created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ProfileDocument, 0)
	for rows.Next() {
		var item ProfileDocument
		var filePath string
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.UploadID, &filePath, &item.OriginalName, &item.MimeType, &item.SizeBytes, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.URL = filePath
		item.Metadata = map[string]any{}
		if len(metadata) > 0 {
			_ = json.Unmarshal(metadata, &item.Metadata)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetStats(ctx context.Context, userID, role string) (ProfileStats, error) {
	stats := ProfileStats{Currency: "KZT"}
	_ = r.db.QueryRow(ctx, `SELECT balance::float8, currency FROM wallets WHERE user_id=$1`, userID).Scan(&stats.Balance, &stats.Currency)

	switch role {
	case "client":
		_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE client_id=$1 AND deleted_at IS NULL`, userID).Scan(&stats.OrdersTotal)
		_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE client_id=$1 AND deleted_at IS NULL AND status IN ('published','in_progress','payment_pending')`, userID).Scan(&stats.OrdersActive)
		_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE client_id=$1 AND deleted_at IS NULL AND status='completed'`, userID).Scan(&stats.OrdersCompleted)
		_ = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0)::float8 FROM wallet_transactions WHERE user_id=$1 AND direction='debit'`, userID).Scan(&stats.SpentTotal)
	case "executor":
		_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE selected_executor_id=$1 AND deleted_at IS NULL`, userID).Scan(&stats.OrdersTotal)
		_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE selected_executor_id=$1 AND deleted_at IS NULL AND status='in_progress'`, userID).Scan(&stats.OrdersActive)
		_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE selected_executor_id=$1 AND deleted_at IS NULL AND status='completed'`, userID).Scan(&stats.OrdersCompleted)
		_ = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0)::float8 FROM wallet_transactions WHERE user_id=$1 AND direction='credit' AND reason='order_completed'`, userID).Scan(&stats.EarnedTotal)
		_ = r.db.QueryRow(ctx, `SELECT rating_avg::float8, rating_count, profile_views, response_rate FROM executor_profiles WHERE user_id=$1`, userID).Scan(&stats.RatingAvg, &stats.RatingCount, &stats.ProfileViews, &stats.ResponseRate)
	}
	return stats, nil
}
