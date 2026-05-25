package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	devpayments "buhpro/internal/modules/devpayments"
	ordersmodule "buhpro/internal/modules/orders"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidRole  = errors.New("invalid role")
	ErrOrderState   = errors.New("invalid order state")
)

type Service struct {
	orders      *ordersmodule.Service
	confirmator *devpayments.Service
}

func NewService(orders *ordersmodule.Service, confirmator *devpayments.Service) *Service {
	return &Service{orders: orders, confirmator: confirmator}
}

func (s *Service) Create(ctx context.Context, userID, role string, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	if role != "client" {
		return CreatePaymentResponse{}, ErrInvalidRole
	}
	if strings.TrimSpace(req.OrderID) == "" || req.Amount <= 0 {
		return CreatePaymentResponse{}, ErrInvalidInput
	}

	_, tx, charge, err := s.orders.SubmitExpectedAmount(ctx, userID, role, req.OrderID, req.Amount)
	if err != nil {
		return CreatePaymentResponse{}, err
	}
	return CreatePaymentResponse{
		TransactionID:     tx.ID,
		YooKassaPaymentID: charge.TransactionID,
		Status:            tx.Status,
		Amount:            tx.Amount,
		Currency:          tx.Currency,
		ConfirmationURL:   charge.RedirectURL,
	}, nil
}

type YooKassaWebhook struct {
	Type   string              `json:"type"`
	Event  string              `json:"event"`
	Object YooKassaPaymentBody `json:"object"`
}

type YooKassaPaymentBody struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Paid   bool   `json:"paid"`
}

func (s *Service) HandleWebhook(ctx context.Context, payload []byte) error {
	var hook YooKassaWebhook
	if err := json.Unmarshal(payload, &hook); err != nil {
		return fmt.Errorf("parse yookassa webhook: %w", err)
	}
	if hook.Event != "payment.succeeded" {
		return nil
	}
	if strings.TrimSpace(hook.Object.ID) == "" {
		return ErrInvalidInput
	}
	if err := s.confirmator.ConfirmProviderPayment(ctx, "yookassa", hook.Object.ID); err != nil {
		return err
	}
	grantPaidServiceToUser(ctx, hook.Object.ID)
	return nil
}

func grantPaidServiceToUser(_ context.Context, _ string) {
	// TODO: начислить оплаченную услугу пользователю после успешной оплаты.
}
