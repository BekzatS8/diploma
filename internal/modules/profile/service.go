package profile

import (
	"context"

	"buhpro/internal/modules/uploads"
)

type Service struct {
	repo    *Repository
	uploads *uploads.Service
}

func NewService(repo *Repository, uploads *uploads.Service) *Service {
	return &Service{repo: repo, uploads: uploads}
}

func (s *Service) GetCurrentProfile(ctx context.Context, userID, role string) (map[string]any, error) {
	profile, err := s.repo.GetByRole(ctx, userID, role)
	if err != nil {
		return nil, err
	}
	if s.uploads != nil {
		if rawID, ok := profile["avatar_upload_id"]; ok && rawID != nil {
			if uploadID, ok := rawID.(*string); ok && uploadID != nil && *uploadID != "" {
				if item, err := s.uploads.GetByID(ctx, *uploadID); err == nil {
					profile["avatar_url"] = s.uploads.URL(item)
				}
			}
		}
	}
	return profile, nil
}

func (s *Service) UpdateCurrentProfile(ctx context.Context, userID, role string, req UpdateProfileRequest) error {
	if req.YearsExperience != nil && *req.YearsExperience < 0 {
		v := 0
		req.YearsExperience = &v
	}
	return s.repo.PatchByRole(ctx, userID, role, req)
}

func (s *Service) SetAvatar(ctx context.Context, userID, role, uploadID string) error {
	item, err := s.uploads.GetByID(ctx, uploadID)
	if err != nil {
		return err
	}
	if role != "admin" && item.AuthorID != userID {
		return uploads.ErrForbidden
	}
	return s.repo.SetAvatarUploadID(ctx, userID, role, uploadID)
}

func (s *Service) ClearAvatar(ctx context.Context, userID, role string) error {
	return s.repo.ClearAvatarUploadID(ctx, userID, role)
}
