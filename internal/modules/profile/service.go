package profile

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetCurrentProfile(ctx context.Context, userID, role string) (map[string]any, error) {
	return s.repo.GetByRole(ctx, userID, role)
}

func (s *Service) UpdateCurrentProfile(ctx context.Context, userID, role string, req UpdateProfileRequest) error {
	if req.YearsExperience != nil && *req.YearsExperience < 0 {
		v := 0
		req.YearsExperience = &v
	}
	return s.repo.PatchByRole(ctx, userID, role, req)
}
