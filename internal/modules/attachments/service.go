package attachments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"buhpro/internal/modules/uploads"

	"github.com/google/uuid"
)

type Service struct {
	repo    *Repository
	uploads *uploads.Service
}

func NewService(repo *Repository, uploads *uploads.Service) *Service {
	return &Service{repo: repo, uploads: uploads}
}

func (s *Service) Attach(ctx context.Context, uploadIDs []string, callerID, role string, targetType TargetType, targetID string, metadata Metadata) ([]AttachmentView, error) {
	if role == "" || callerID == "" || len(uploadIDs) == 0 || !isValidTarget(targetType) {
		return nil, ErrInvalidInput
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return nil, ErrInvalidInput
	}

	count, err := s.repo.CountByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}

	views := make([]AttachmentView, 0, len(uploadIDs))
	for i, uploadID := range uploadIDs {
		if _, err := uuid.Parse(uploadID); err != nil {
			return nil, ErrInvalidInput
		}
		upload, err := s.uploads.GetByID(ctx, uploadID)
		if err != nil {
			return nil, err
		}
		if role != "admin" && upload.AuthorID != callerID {
			return nil, ErrForbidden
		}

		item := Attachment{
			ID:         uuid.NewString(),
			UploadID:   uploadID,
			TargetType: targetType,
			TargetID:   targetID,
			SortOrder:  count + i,
			Metadata:   metadata,
			CreatedAt:  time.Now(),
		}
		if err := s.repo.Create(ctx, item); err != nil {
			return nil, fmt.Errorf("create attachment: %w", err)
		}
		views = append(views, s.toView(item, upload))
	}

	return views, nil
}

func (s *Service) ListByTarget(ctx context.Context, targetType TargetType, targetID string) ([]AttachmentView, error) {
	if !isValidTarget(targetType) {
		return nil, ErrInvalidInput
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return nil, ErrInvalidInput
	}

	items, err := s.repo.ListByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	views := make([]AttachmentView, 0, len(items))
	for _, item := range items {
		upload, err := s.uploads.GetByID(ctx, item.UploadID)
		if err != nil {
			if errors.Is(err, uploads.ErrNotFound) {
				continue
			}
			return nil, err
		}
		views = append(views, s.toView(item, upload))
	}
	return views, nil
}

func (s *Service) Delete(ctx context.Context, id, callerID, role string) error {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	upload, err := s.uploads.GetByID(ctx, item.UploadID)
	if err != nil {
		return err
	}
	if role != "admin" && upload.AuthorID != callerID {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) Reorder(ctx context.Context, ids []string, callerID, role string) error {
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	for i, id := range ids {
		item, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		upload, err := s.uploads.GetByID(ctx, item.UploadID)
		if err != nil {
			return err
		}
		if role != "admin" && upload.AuthorID != callerID {
			return ErrForbidden
		}
		if err := s.repo.UpdateSortOrder(ctx, id, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) toView(item Attachment, upload uploads.Upload) AttachmentView {
	return AttachmentView{
		ID:           item.ID,
		UploadID:     item.UploadID,
		TargetType:   item.TargetType,
		TargetID:     item.TargetID,
		SortOrder:    item.SortOrder,
		Metadata:     item.Metadata,
		CreatedAt:    item.CreatedAt,
		URL:          s.uploads.URL(upload),
		OriginalName: upload.OriginalName,
		MimeType:     upload.MimeType,
		SizeBytes:    upload.SizeBytes,
	}
}

func isValidTarget(targetType TargetType) bool {
	switch targetType {
	case TargetProfileDocument, TargetOrderAttachment, TargetResponseAttachment, TargetReviewAttachment, TargetChatAttachment, TargetCourseMaterial:
		return true
	default:
		return false
	}
}
