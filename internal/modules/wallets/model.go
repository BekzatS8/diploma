package wallets

import "time"

type Wallet struct {
	UserID    string    `json:"user_id"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transaction struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Direction string    `json:"direction"`
	Currency  string    `json:"currency"`
	Reason    string    `json:"reason"`
	OrderID   *string   `json:"order_id,omitempty"`
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
