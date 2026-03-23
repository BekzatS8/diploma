package profile

type UpdateProfileRequest struct {
	ProfileName     *string `json:"profile_name"`
	Phone           *string `json:"phone"`
	Bio             *string `json:"bio"`
	Expertise       *string `json:"expertise"`
	CompanyName     *string `json:"company_name"`
	About           *string `json:"about"`
	YearsExperience *int    `json:"years_experience"`
}
