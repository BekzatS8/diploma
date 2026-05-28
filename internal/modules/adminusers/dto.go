package adminusers

type ExecutorListItem struct {
	UserID      string  `json:"user_id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	IsActive    bool    `json:"is_active"`
	IsCoach     bool    `json:"is_coach"`
	RatingAvg   float64 `json:"rating_avg"`
	RatingCount int     `json:"rating_count"`
	CreatedAt   string  `json:"created_at"`
}

type ExecutorListResponse struct {
	Items    []ExecutorListItem `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}

type PromoteCoachResponse struct {
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	IsCoach bool   `json:"is_coach"`
	Message string `json:"message"`
}

type RevokeCoachResponse struct {
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	IsCoach bool   `json:"is_coach"`
	Message string `json:"message"`
}
