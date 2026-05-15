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
	UploadID string `json:"upload_id" binding:"required"`
}
