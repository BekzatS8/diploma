package auth

type RegisterRequest struct {
	Email           string   `json:"email" binding:"required,email"`
	Password        string   `json:"password" binding:"required"`
	Role            string   `json:"role" binding:"required"`
	ProfileName     string   `json:"profile_name"`
	Phone           string   `json:"phone"`
	ClientType      string   `json:"client_type"`
	TaxNumber       string   `json:"tax_number"`
	ContactName     string   `json:"contact_name"`
	ContactPosition string   `json:"contact_position"`
	Address         string   `json:"address"`
	About           string   `json:"about"`
	Website         string   `json:"website"`
	FirstName       string   `json:"first_name"`
	LastName        string   `json:"last_name"`
	MiddleName      string   `json:"middle_name"`
	IIN             string   `json:"iin"`
	City            string   `json:"city"`
	ExperienceLevel string   `json:"experience_level"`
	Specializations []string `json:"specializations"`
	Education       string   `json:"education"`
	WorkFormat      string   `json:"work_format"`
	HourlyRate      *float64 `json:"hourly_rate"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordResponse struct {
	Message        string `json:"message"`
	ResetURL       string `json:"reset_url,omitempty"`
	EmailSent      bool   `json:"email_sent,omitempty"`
	MailConfigured bool   `json:"mail_configured,omitempty"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type AuthResponse struct {
	UserID             string `json:"user_id"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	VerificationStatus string `json:"verification_status,omitempty"`
	AccessToken        string `json:"access_token,omitempty"`
	RefreshToken       string `json:"refresh_token,omitempty"`
}
