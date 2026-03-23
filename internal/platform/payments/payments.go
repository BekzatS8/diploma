package payments

import "context"

type ChargeRequest struct {
	OrderID      string
	AmountCents  int64
	CurrencyCode string
	Description  string
}

type ChargeResponse struct {
	TransactionID string
	Status        string
	RedirectURL   string
}

type Provider interface {
	CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResponse, error)
	VerifyWebhook(ctx context.Context, payload []byte, signature string) error
}

type MockProvider struct{}

func NewMock() Provider {
	return &MockProvider{}
}

func (m *MockProvider) CreateCharge(_ context.Context, req ChargeRequest) (ChargeResponse, error) {
	return ChargeResponse{
		TransactionID: "mock_txn_" + req.OrderID,
		Status:        "pending",
		RedirectURL:   "https://mock-payments.local/checkout/" + req.OrderID,
	}, nil
}

func (m *MockProvider) VerifyWebhook(_ context.Context, _ []byte, _ string) error {
	return nil
}
