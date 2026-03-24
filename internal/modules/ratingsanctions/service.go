package ratingsanctions

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) HasActiveResponseRestriction(ctx context.Context, executorID string) (bool, error) {
	return s.repo.HasActiveLowRating(ctx, executorID)
}

func (s *Service) RecalculateAndApplyTx(ctx context.Context, tx pgx.Tx, executorID string) (RecalculateResult, error) {
	return s.repo.RecalculateAndApply(ctx, tx, executorID)
}

func (s *Service) GetRating(ctx context.Context, executorID string) (RatingInfo, error) {
	return s.repo.GetRating(ctx, executorID)
}

func (s *Service) ListMySanctions(ctx context.Context, userID, role string) ([]Sanction, error) {
	if role != "executor" && role != "admin" {
		return nil, ErrForbidden
	}
	if role == "admin" {
		return s.repo.ListMy(ctx, userID)
	}
	return s.repo.ListMy(ctx, userID)
}

func (s *Service) ListAdmin(ctx context.Context, role string, page, size int) ([]Sanction, int64, error) {
	if role != "admin" {
		return nil, 0, ErrForbidden
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListAdmin(ctx, size, (page-1)*size)
}

func (s *Service) GetAdmin(ctx context.Context, role, id string) (Sanction, error) {
	if role != "admin" {
		return Sanction{}, ErrForbidden
	}
	item, err := s.repo.GetSanction(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Sanction{}, ErrNotFound
		}
		return Sanction{}, err
	}
	return item, nil
}

func (s *Service) Lift(ctx context.Context, role, id, actorID string) error {
	if role != "admin" {
		return ErrForbidden
	}
	return s.repo.Lift(ctx, id, actorID)
}
