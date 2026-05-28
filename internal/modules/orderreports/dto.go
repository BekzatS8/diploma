package orderreports

import "time"

type CreateReportRequest struct {
	Reason string `json:"reason" binding:"required,min=10,max=2000"`
}

type CreateReportResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ReviewRequest struct {
	Notes string `json:"notes" binding:"max=1000"`
}

type ReportView struct {
	ID             string     `json:"id"`
	OrderID        string     `json:"order_id"`
	ReporterID     string     `json:"reporter_id"`
	ReporterEmail  string     `json:"reporter_email"`
	ReporterName   string     `json:"reporter_name"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	AdminNotes     *string    `json:"admin_notes,omitempty"`
	OrderTitle     string     `json:"order_title"`
	OrderDescription string   `json:"order_description"`
	OrderBudget    float64    `json:"order_budget"`
	OrderCurrency  string     `json:"order_currency"`
	OrderStatus    string     `json:"order_status"`
	CreatedAt      time.Time  `json:"created_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
}

type ListResponse struct {
	Items    []ReportView `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Total    int64        `json:"total"`
}

type ReviewResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
