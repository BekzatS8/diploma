package responses

import "time"

type CreateResponseRequest struct {
	CoverLetter    *string  `json:"cover_letter" binding:"omitempty,max=5000"`
	ProposedAmount *float64 `json:"proposed_amount" binding:"omitempty,gt=0"`
	Currency       *string  `json:"currency" binding:"omitempty,len=3,alpha"`
}

type UpdateResponseRequest struct {
	CoverLetter    *string  `json:"cover_letter" binding:"omitempty,max=5000"`
	ProposedAmount *float64 `json:"proposed_amount" binding:"omitempty,gt=0"`
	Currency       *string  `json:"currency" binding:"omitempty,len=3,alpha"`
}

type ListQuery struct {
	Status   string
	Page     int
	PageSize int
}

type ResponseView struct {
	ID             string     `json:"id"`
	OrderID        string     `json:"order_id"`
	ExecutorID     string     `json:"executor_id,omitempty"`
	CoverLetter    *string    `json:"cover_letter,omitempty"`
	ProposedAmount *float64   `json:"proposed_amount,omitempty"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	IsPaid         bool       `json:"is_paid"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	OrderTitle     *string    `json:"order_title,omitempty"`
}

type ListResponse struct {
	Items    []ResponseView `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int64          `json:"total"`
}

type SubmitResponsePayload struct {
	Response ResponseView          `json:"response"`
	Payment  SubmitPaymentNextStep `json:"payment"`
}

type SubmitPaymentNextStep struct {
	TransactionID string  `json:"transaction_id"`
	Provider      string  `json:"provider"`
	Status        string  `json:"status"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	CheckoutURL   string  `json:"checkout_url"`
	ProviderRef   string  `json:"provider_ref"`
}
