package profile

import (
	"context"
	"strings"

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
	docs, err := s.repo.ListProfileDocuments(ctx, userID)
	if err == nil {
		if s.uploads != nil {
			for i := range docs {
				docs[i].URL = s.uploads.URL(uploads.Upload{FilePath: docs[i].URL})
			}
		}
		profile["documents"] = docs
	}
	stats, err := s.repo.GetStats(ctx, userID, role)
	if err == nil {
		profile["stats"] = stats
		profile["achievements"] = achievementsFor(role, stats)
	}
	return profile, nil
}

func (s *Service) UpdateCurrentProfile(ctx context.Context, userID, role string, req UpdateProfileRequest) error {
	if req.YearsExperience != nil && *req.YearsExperience < 0 {
		v := 0
		req.YearsExperience = &v
	}
	if req.HourlyRate != nil && *req.HourlyRate < 0 {
		v := 0.0
		req.HourlyRate = &v
	}
	req.Website = normalizeStringPtr(req.Website)
	req.Phone = normalizeStringPtr(req.Phone)
	req.About = normalizeStringPtr(req.About)
	req.Bio = normalizeStringPtr(req.Bio)
	if req.IIN != nil {
		v := strings.TrimSpace(*req.IIN)
		if v != "" && len(v) != 12 {
			return uploads.ErrInvalidInput
		}
		req.IIN = &v
	}
	req.Specializations = normalizeSpecializations(req.Specializations)
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

func normalizeSpecializations(items []string) []string {
	if items == nil {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizeStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func achievementsFor(role string, stats ProfileStats) []ProfileAchievement {
	items := make([]ProfileAchievement, 0)
	if stats.OrdersCompleted > 0 {
		items = append(items, ProfileAchievement{
			Code:        "first_completed_order",
			Title:       "First completed order",
			Description: "At least one order has been completed.",
		})
	}
	if role == "executor" {
		if stats.RatingAvg >= 4.5 && stats.RatingCount > 0 {
			items = append(items, ProfileAchievement{
				Code:        "top_executor",
				Title:       "Top executor",
				Description: "Rating is 4.5 or higher.",
			})
		}
		if stats.OrdersCompleted >= 10 {
			items = append(items, ProfileAchievement{
				Code:        "reliable_partner",
				Title:       "Reliable partner",
				Description: "10 or more completed orders.",
			})
		}
		if stats.ResponseRate >= 80 {
			items = append(items, ProfileAchievement{
				Code:        "fast_response",
				Title:       "Fast response",
				Description: "Response rate is 80% or higher.",
			})
		}
	}
	if role == "client" && stats.SpentTotal > 0 {
		items = append(items, ProfileAchievement{
			Code:        "active_customer",
			Title:       "Active customer",
			Description: "Has paid for order placement or escrow.",
		})
	}
	return items
}
