package selection

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrInvalidState    = errors.New("invalid state")
	ErrAlreadySelected = errors.New("already selected")
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) SelectResponse(ctx context.Context, orderID, responseID, userID, role string) error {
	ok, err := s.repo.IsOrderOwnerOrAdmin(ctx, orderID, userID, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !ok {
		return ErrForbidden
	}
	err = s.repo.SelectResponse(ctx, orderID, responseID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Service) GetSelection(ctx context.Context, orderID, userID, role string) (Selection, error) {
	ok, err := s.repo.IsOrderOwnerOrAdmin(ctx, orderID, userID, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Selection{}, ErrNotFound
		}
		return Selection{}, err
	}
	if !ok {
		return Selection{}, ErrForbidden
	}
	item, err := s.repo.GetSelection(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Selection{}, ErrNotFound
		}
		return Selection{}, err
	}
	return item, nil
}

func (s *Service) Complete(ctx context.Context, orderID, userID, role string) error {
	ok, err := s.repo.IsOrderOwnerOrAdmin(ctx, orderID, userID, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !ok {
		return ErrForbidden
	}
	err = s.repo.Complete(ctx, orderID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Service) Reopen(ctx context.Context, orderID, userID, role string) error {
	ok, err := s.repo.IsOrderOwnerOrAdmin(ctx, orderID, userID, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !ok {
		return ErrForbidden
	}
	err = s.repo.Reopen(ctx, orderID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
