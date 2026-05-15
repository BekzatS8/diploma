package auth

import "time"

type User struct {
	ID                          string
	Email                       string
	PasswordHash                string
	Role                        string
	IsActive                    bool
	VerificationStatus          string
	VerificationSubmittedAt     *time.Time
	VerificationReviewedAt      *time.Time
	VerificationReviewedBy      *string
	VerificationRejectionReason *string
	CreatedAt                   time.Time
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}
