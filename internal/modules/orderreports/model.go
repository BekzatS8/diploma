package orderreports

import "time"

type Status string

const (
	StatusPending      Status = "pending"
	StatusDismissed    Status = "dismissed"
	StatusOrderRemoved Status = "order_removed"
)

type Report struct {
	ID           string
	OrderID      string
	ReporterID   string
	Reason       string
	Status       Status
	AdminID      *string
	AdminNotes   *string
	CreatedAt    time.Time
	ReviewedAt   *time.Time
	OrderTitle   string
	OrderDesc    string
	OrderBudget  float64
	OrderCurrency string
	OrderStatus  string
	ReporterEmail string
	ReporterName  string
}
