package notifications

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) EmitInApp(ctx context.Context, userID, notificationType string, payload map[string]any) (Notification, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(notificationType) == "" {
		return Notification{}, ErrInvalidInput
	}
	return s.repo.Create(ctx, CreateParams{
		UserID:  userID,
		Type:    notificationType,
		Channel: ChannelInApp,
		Status:  StatusSent,
		Payload: payload,
	})
}

func (s *Service) ListMy(ctx context.Context, userID, role string, q ListQuery) ([]Notification, int64, error) {
	if role == "" {
		return nil, 0, ErrForbidden
	}
	normalizeListQuery(&q)
	items, total, err := s.repo.ListMy(ctx, userID, q)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetMyByID(ctx context.Context, id, userID, role string) (Notification, error) {
	if role == "" {
		return Notification{}, ErrForbidden
	}
	item, err := s.repo.GetMyByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotFound
		}
		return Notification{}, err
	}
	return item, nil
}

func (s *Service) MarkRead(ctx context.Context, id, userID, role string) (Notification, error) {
	if role == "" {
		return Notification{}, ErrForbidden
	}
	item, err := s.repo.MarkRead(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotFound
		}
		return Notification{}, err
	}
	return item, nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID, role string) (int64, error) {
	if role == "" {
		return 0, ErrForbidden
	}
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) ListAdmin(ctx context.Context, role string, q ListQuery) ([]Notification, int64, error) {
	if role != "admin" {
		return nil, 0, ErrForbidden
	}
	normalizeListQuery(&q)
	return s.repo.ListAdmin(ctx, q)
}

func (s *Service) GetAdminByID(ctx context.Context, role, id string) (Notification, error) {
	if role != "admin" {
		return Notification{}, ErrForbidden
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotFound
		}
		return Notification{}, err
	}
	return item, nil
}

func normalizeListQuery(q *ListQuery) {
	q.Type = strings.TrimSpace(q.Type)
	q.Status = strings.TrimSpace(q.Status)
	q.Channel = strings.TrimSpace(q.Channel)
	q.UserID = strings.TrimSpace(q.UserID)
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
}
