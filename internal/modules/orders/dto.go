package orders

import "time"

type CreateOrderRequest struct {
	Title        string     `json:"title" binding:"required,min=3,max=200"`
	Description  string     `json:"description" binding:"required,min=10,max=5000"`
	CategoryID   *int64     `json:"category_id"`
	CategorySlug *string    `json:"category_slug"`
	BudgetAmount float64    `json:"budget_amount" binding:"required,gt=0"`
	Currency     *string    `json:"currency"`
	DeadlineAt   *time.Time `json:"deadline_at"`
	Region       *string    `json:"region"`
	Promotions   []string   `json:"promotions"`
}

type UpdateOrderRequest struct {
	Title        *string    `json:"title"`
	Description  *string    `json:"description"`
	CategoryID   *int64     `json:"category_id"`
	CategorySlug *string    `json:"category_slug"`
	BudgetAmount *float64   `json:"budget_amount"`
	Currency     *string    `json:"currency"`
	DeadlineAt   *time.Time `json:"deadline_at"`
	Region       *string    `json:"region"`
	Promotions   []string   `json:"promotions"`
}

type PublicOrdersQuery struct {
	CategorySlug   string
	BudgetMin      *float64
	BudgetMax      *float64
	DeadlineBefore *time.Time
	Region         string
	Q              string
	Page           int
	PageSize       int
}

type MyOrdersQuery struct {
	Status   string
	Page     int
	PageSize int
}

type SubmitOrderResponse struct {
	Order   OrderResponse         `json:"order"`
	Payment SubmitPaymentNextStep `json:"payment"`
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

type OrderResponse struct {
	ID               string     `json:"id"`
	ClientID         string     `json:"client_id,omitempty"`
	CategoryID       *int64     `json:"category_id,omitempty"`
	CategorySlug     *string    `json:"category_slug,omitempty"`
	CategoryName     *string    `json:"category_name,omitempty"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	BudgetAmount     float64    `json:"budget_amount"`
	Currency         string     `json:"currency"`
	DeadlineAt       *time.Time `json:"deadline_at,omitempty"`
	Region           *string    `json:"region,omitempty"`
	Promotions       []string   `json:"promotions"`
	PostingFee       float64    `json:"posting_fee"`
	PromotionFee     float64    `json:"promotion_fee"`
	EscrowAmount     float64    `json:"escrow_amount"`
	TotalCharge      float64    `json:"total_charge"`
	PaymentStatus    string     `json:"payment_status"`
	PromotedUntil    *time.Time `json:"promoted_until,omitempty"`
	PinnedUntil      *time.Time `json:"pinned_until,omitempty"`
	HighlightedUntil *time.Time `json:"highlighted_until,omitempty"`
	Status           string     `json:"status"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type OrdersListResponse struct {
	Items    []OrderResponse `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
}
