package devpayments

import (
	"context"
	"errors"

	notifications "buhpro/internal/modules/notifications"
)

var (
	ErrDevDisabled = errors.New("dev payment endpoints disabled")
)

type Service struct {
	repo     *Repository
	enabled  bool
	notifier *notifications.Service
}

func NewService(repo *Repository, enabled bool, notifier *notifications.Service) *Service {
	return &Service{repo: repo, enabled: enabled, notifier: notifier}
}

func (s *Service) Confirm(ctx context.Context, transactionID string) error {
	if !s.enabled {
		return ErrDevDisabled
	}
	result, err := s.repo.Confirm(ctx, transactionID)
	if err != nil {
		return err
	}
	s.emitNotifications(ctx, result)
	return nil
}

func (s *Service) ConfirmProviderPayment(ctx context.Context, provider, providerRef string) error {
	result, err := s.repo.ConfirmByProviderRef(ctx, provider, providerRef)
	if err != nil {
		return err
	}
	s.emitNotifications(ctx, result)
	return nil
}

func (s *Service) Fail(ctx context.Context, transactionID string) error {
	if !s.enabled {
		return ErrDevDisabled
	}
	return s.repo.Fail(ctx, transactionID)
}

func (s *Service) emitNotifications(ctx context.Context, result ConfirmResult) {
	if s.notifier == nil {
		return
	}
	if result.OrderPublishedForClient != nil {
		_, _ = s.notifier.EmitInApp(ctx, result.OrderPublishedForClient.ClientID, notifications.TypeOrderPublished, map[string]any{
			"order_id": result.OrderPublishedForClient.OrderID,
		})
	}
	if result.ResponseSubmittedClient != nil {
		_, _ = s.notifier.EmitInApp(ctx, result.ResponseSubmittedClient.ClientID, notifications.TypeResponseSubmitted, map[string]any{
			"order_id":    result.ResponseSubmittedClient.OrderID,
			"response_id": result.ResponseSubmittedClient.ResponseID,
		})
	}
}
