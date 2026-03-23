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
