package profile

type UpdateProfileRequest struct {
	ProfileName     *string  `json:"profile_name" binding:"omitempty,min=1,max=200"`
	Phone           *string  `json:"phone" binding:"omitempty,max=32"`
	Bio             *string  `json:"bio" binding:"omitempty,max=5000"`
	Expertise       *string  `json:"expertise" binding:"omitempty,max=500"`
	CompanyName     *string  `json:"company_name" binding:"omitempty,max=200"`
	ClientType      *string  `json:"client_type" binding:"omitempty,oneof=too ip representative"`
	TaxNumber       *string  `json:"tax_number" binding:"omitempty,max=64"`
	ContactName     *string  `json:"contact_name" binding:"omitempty,max=200"`
	ContactPosition *string  `json:"contact_position" binding:"omitempty,max=200"`
	Address         *string  `json:"address" binding:"omitempty,max=500"`
	About           *string  `json:"about" binding:"omitempty,max=5000"`
	Website         *string  `json:"website" binding:"omitempty,max=2048"`
	YearsExperience *int     `json:"years_experience" binding:"omitempty,gte=0,lte=80"`
	FirstName       *string  `json:"first_name" binding:"omitempty,max=100"`
	LastName        *string  `json:"last_name" binding:"omitempty,max=100"`
	MiddleName      *string  `json:"middle_name" binding:"omitempty,max=100"`
	IIN             *string  `json:"iin" binding:"omitempty,len=12,numeric"`
	City            *string  `json:"city" binding:"omitempty,max=100"`
	ExperienceLevel *string  `json:"experience_level" binding:"omitempty,max=100"`
	Specializations []string `json:"specializations" binding:"omitempty,dive,min=1,max=100"`
	Education       *string  `json:"education" binding:"omitempty,max=1000"`
	WorkFormat      *string  `json:"work_format" binding:"omitempty,max=100"`
	HourlyRate      *float64 `json:"hourly_rate" binding:"omitempty,gte=0,lte=10000000"`
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
	Balance          float64 `json:"balance"`
	Currency         string  `json:"currency"`
	OrdersTotal      int64   `json:"orders_total"`
	OrdersActive     int64   `json:"orders_active"`
	OrdersCompleted  int64   `json:"orders_completed"`
	EarnedTotal      float64 `json:"earned_total"`
	SpentTotal       float64 `json:"spent_total"`
	RatingAvg        float64 `json:"rating_avg"`
	RatingCount      int64   `json:"rating_count"`
	ProfileViews     int64   `json:"profile_views"`
	ResponseRate     int64   `json:"response_rate"`
	CoursesPublished int64   `json:"courses_published"`
	CourseStudents   int64   `json:"course_students"`
}

type ProfileAchievement struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
