package profile

type UpdateProfileRequest struct {
	ProfileName     *string  `json:"profile_name"`
	Phone           *string  `json:"phone"`
	Bio             *string  `json:"bio"`
	Expertise       *string  `json:"expertise"`
	CompanyName     *string  `json:"company_name"`
	ClientType      *string  `json:"client_type"`
	TaxNumber       *string  `json:"tax_number"`
	ContactName     *string  `json:"contact_name"`
	ContactPosition *string  `json:"contact_position"`
	Address         *string  `json:"address"`
	About           *string  `json:"about"`
	Website         *string  `json:"website"`
	YearsExperience *int     `json:"years_experience"`
	FirstName       *string  `json:"first_name"`
	LastName        *string  `json:"last_name"`
	MiddleName      *string  `json:"middle_name"`
	IIN             *string  `json:"iin"`
	City            *string  `json:"city"`
	ExperienceLevel *string  `json:"experience_level"`
	Specializations []string `json:"specializations"`
	Education       *string  `json:"education"`
	WorkFormat      *string  `json:"work_format"`
	HourlyRate      *float64 `json:"hourly_rate"`
}

type SetAvatarRequest struct {
	UploadID string `json:"upload_id" binding:"required,uuid"`
}

type ProfileDocument struct {
	ID           string         `json:"id"`
	UploadID     string         `json:"upload_id"`
	URL          string         `json:"url"`
	OriginalName string         `json:"original_name"`
	MimeType     string         `json:"mime_type"`
	SizeBytes    int64          `json:"size_bytes"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    string         `json:"created_at"`
}

type ProfileStats struct {
	Balance         float64 `json:"balance"`
	Currency        string  `json:"currency"`
	OrdersTotal     int64   `json:"orders_total"`
	OrdersActive    int64   `json:"orders_active"`
	OrdersCompleted int64   `json:"orders_completed"`
	EarnedTotal     float64 `json:"earned_total"`
	SpentTotal      float64 `json:"spent_total"`
	RatingAvg       float64 `json:"rating_avg"`
	RatingCount     int64   `json:"rating_count"`
	ProfileViews    int64   `json:"profile_views"`
	ResponseRate    int64   `json:"response_rate"`
}

type ProfileAchievement struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
