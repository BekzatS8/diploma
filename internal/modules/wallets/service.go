package wallets

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct {
	repo            *Repository
	defaultCurrency string
}

func NewService(repo *Repository, defaultCurrency string) *Service {
	return &Service{repo: repo, defaultCurrency: strings.ToUpper(strings.TrimSpace(defaultCurrency))}
}

func (s *Service) Get(ctx context.Context, userID, role, targetUserID string) (WalletResponse, error) {
	if strings.TrimSpace(targetUserID) == "" {
		targetUserID = userID
	}
	if role != "admin" && targetUserID != userID {
		return WalletResponse{}, ErrForbidden
	}
	if _, err := uuid.Parse(targetUserID); err != nil {
		return WalletResponse{}, ErrInvalidInput
	}
	wallet, err := s.repo.Ensure(ctx, targetUserID, s.defaultCurrency)
	if err != nil {
		return WalletResponse{}, err
	}
	items, err := s.repo.ListTransactions(ctx, targetUserID, 50)
	if err != nil {
		return WalletResponse{}, err
	}
	return WalletResponse{Wallet: wallet, Transactions: items}, nil
}

func (s *Service) Credit(ctx context.Context, actorID, role, targetUserID string, amount float64, reason string) (Wallet, error) {
	if role != "admin" {
		return Wallet{}, ErrForbidden
	}
	if _, err := uuid.Parse(targetUserID); err != nil {
		return Wallet{}, ErrInvalidInput
	}
	if amount <= 0 || amount > 100000000 {
		return Wallet{}, ErrInvalidInput
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 100 {
		return Wallet{}, ErrInvalidInput
	}
	if reason == "" {
		reason = "admin_credit"
	}
	return s.repo.Credit(ctx, targetUserID, actorID, amount, s.defaultCurrency, reason)
}
