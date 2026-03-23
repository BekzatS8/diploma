package devpayments

import (
	"context"
	"errors"
)

var (
	ErrDevDisabled = errors.New("dev payment endpoints disabled")
)

type Service struct {
	repo    *Repository
	enabled bool
}

func NewService(repo *Repository, enabled bool) *Service {
	return &Service{repo: repo, enabled: enabled}
}

func (s *Service) Confirm(ctx context.Context, transactionID string) error {
	if !s.enabled {
		return ErrDevDisabled
	}
	return s.repo.Confirm(ctx, transactionID)
}

func (s *Service) Fail(ctx context.Context, transactionID string) error {
	if !s.enabled {
		return ErrDevDisabled
	}
	return s.repo.Fail(ctx, transactionID)
}
