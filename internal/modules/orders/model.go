package orders

import "time"

type Order struct {
	ID                 string
	ClientID           string
	CategoryID         *int64
	CategorySlug       *string
	CategoryName       *string
	Title              string
	Description        string
	BudgetAmount       float64
	Currency           string
	DeadlineAt         *time.Time
	Region             *string
	PromotionOptions   []string
	PostingFee         float64
	PromotionFee       float64
	EscrowAmount       float64
	TotalCharge        float64
	PaymentStatus      string
	PromotedUntil      *time.Time
	PinnedUntil        *time.Time
	HighlightedUntil   *time.Time
	ExecutorPaidAt     *time.Time
	Status             string
	SelectedExecutorID *string
	PublishedAt        *time.Time
	CompletedAt        *time.Time
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

type PaymentTransaction struct {
	ID                    string
	OrderID               *string
	Provider              string
	ProviderTransactionID *string
	Amount                float64
	Currency              string
	Status                string
	InitiatedAt           time.Time
}
