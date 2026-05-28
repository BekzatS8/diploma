package adminusers

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListExecutors(ctx context.Context, q string, page, pageSize int) (ExecutorListResponse, error) {
	items, total, err := s.repo.ListExecutors(ctx, q, page, pageSize)
	if err != nil {
		return ExecutorListResponse{}, err
	}
	if items == nil {
		items = []ExecutorListItem{}
	}
	return ExecutorListResponse{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (s *Service) PromoteExecutorToCoach(ctx context.Context, userID string) (PromoteCoachResponse, error) {
	if err := s.repo.PromoteExecutorToCoach(ctx, userID); err != nil {
		return PromoteCoachResponse{}, err
	}
	return PromoteCoachResponse{
		UserID:  userID,
		Role:    "executor",
		IsCoach: true,
		Message: "Исполнителю выданы права коуча. Роль исполнителя сохранена — нужен повторный вход.",
	}, nil
}

func (s *Service) RevokeCoachFromExecutor(ctx context.Context, userID string) (RevokeCoachResponse, error) {
	if err := s.repo.RevokeCoachFromExecutor(ctx, userID); err != nil {
		return RevokeCoachResponse{}, err
	}
	return RevokeCoachResponse{
		UserID:  userID,
		Role:    "executor",
		IsCoach: false,
		Message: "Права коуча сняты. Исполнитель сохранён — нужен повторный вход.",
	}, nil
}
