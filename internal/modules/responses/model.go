package responses

import "time"

type Response struct {
	ID             string
	OrderID        string
	ExecutorID     string
	OrderClientID  string
	OrderStatus    string
	CoverLetter    *string
	ProposedAmount *float64
	Currency       string
	Status         string
	IsPaid         bool
	PaidAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
	OrderTitle     *string
}

type PaymentTransaction struct {
	ID                    string
	ResponseID            *string
	Provider              string
	ProviderTransactionID *string
	Amount                float64
	Currency              string
	Status                string
	InitiatedAt           time.Time
}
