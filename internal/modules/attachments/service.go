package attachments

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type targetAction int

const (
	targetActionRead targetAction = iota
	targetActionWrite
)

func (s *Service) Attach(ctx context.Context, uploadIDs []string, callerID, role string, targetType TargetType, targetID string, metadata Metadata) ([]AttachmentView, error) {
	if role == "" || callerID == "" || len(uploadIDs) == 0 || !isValidTarget(targetType) {
		return nil, ErrInvalidInput
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return nil, ErrInvalidInput
	}
	if err := s.ensureTargetAccess(ctx, targetType, targetID, callerID, role, targetActionWrite); err != nil {
		return nil, err
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

func (s *Service) ListByTarget(ctx context.Context, targetType TargetType, targetID, callerID, role string) ([]AttachmentView, error) {
	if role == "" || callerID == "" || !isValidTarget(targetType) {
		return nil, ErrInvalidInput
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return nil, ErrInvalidInput
	}
	if err := s.ensureTargetAccess(ctx, targetType, targetID, callerID, role, targetActionRead); err != nil {
		return nil, err
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
	if role == "" || callerID == "" {
		return ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidInput
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ensureTargetAccess(ctx, item.TargetType, item.TargetID, callerID, role, targetActionWrite); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) Reorder(ctx context.Context, ids []string, callerID, role string) error {
	if role == "" || callerID == "" {
		return ErrForbidden
	}
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	var targetType TargetType
	var targetID string
	items := make([]Attachment, 0, len(ids))
	for i, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return ErrInvalidInput
		}
		item, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if i == 0 {
			targetType = item.TargetType
			targetID = item.TargetID
		} else if item.TargetType != targetType || item.TargetID != targetID {
			return ErrInvalidInput
		}
		items = append(items, item)
	}
	if err := s.ensureTargetAccess(ctx, targetType, targetID, callerID, role, targetActionWrite); err != nil {
		return err
	}
	for i, item := range items {
		if err := s.repo.UpdateSortOrder(ctx, item.ID, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ensureTargetAccess(ctx context.Context, targetType TargetType, targetID, callerID, role string, action targetAction) error {
	if role == "" || callerID == "" {
		return ErrForbidden
	}
	if role == "admin" {
		return s.ensureTargetExists(ctx, targetType, targetID)
	}

	switch targetType {
	case TargetOrderAttachment:
		target, err := s.repo.GetOrderTarget(ctx, targetID)
		if err != nil {
			return err
		}
		if target.ClientID == callerID {
			return nil
		}
		if action == targetActionRead && target.SelectedExecutorID != nil && *target.SelectedExecutorID == callerID {
			return nil
		}
		return ErrForbidden
	case TargetResponseAttachment:
		target, err := s.repo.GetResponseTarget(ctx, targetID)
		if err != nil {
			return err
		}
		if target.ExecutorID == callerID {
			return nil
		}
		if action == targetActionRead && target.OrderClientID == callerID && target.IsPaid && isClientVisibleResponseStatus(target.Status) {
			return nil
		}
		return ErrForbidden
	case TargetChatAttachment:
		ok, err := s.repo.IsChatTargetParticipant(ctx, targetID, callerID)
		if err != nil {
			return err
		}
		if !ok {
			exists, err := s.repo.ChatTargetExists(ctx, targetID)
			if err != nil {
				return err
			}
			if !exists {
				return ErrNotFound
			}
			return ErrForbidden
		}
		return nil
	case TargetCourseMaterial:
		target, err := s.repo.GetCourseTarget(ctx, targetID)
		if err != nil {
			return err
		}
		if isCourseOwner(target, callerID) {
			return nil
		}
		if action == targetActionRead && role == "executor" && target.Status == "published" {
			return nil
		}
		return ErrForbidden
	case TargetProfileDocument:
		exists, err := s.repo.UserExists(ctx, targetID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		if targetID == callerID {
			return nil
		}
		return ErrForbidden
	case TargetReviewAttachment:
		target, err := s.repo.GetReviewTarget(ctx, targetID)
		if err != nil {
			return err
		}
		if action == targetActionRead {
			return nil
		}
		if target.AuthorID != nil && *target.AuthorID == callerID {
			return nil
		}
		return ErrForbidden
	default:
		return ErrInvalidInput
	}
}

func (s *Service) ensureTargetExists(ctx context.Context, targetType TargetType, targetID string) error {
	switch targetType {
	case TargetOrderAttachment:
		_, err := s.repo.GetOrderTarget(ctx, targetID)
		return err
	case TargetResponseAttachment:
		_, err := s.repo.GetResponseTarget(ctx, targetID)
		return err
	case TargetChatAttachment:
		exists, err := s.repo.ChatTargetExists(ctx, targetID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return nil
	case TargetCourseMaterial:
		_, err := s.repo.GetCourseTarget(ctx, targetID)
		return err
	case TargetProfileDocument:
		exists, err := s.repo.UserExists(ctx, targetID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return nil
	case TargetReviewAttachment:
		_, err := s.repo.GetReviewTarget(ctx, targetID)
		return err
	default:
		return ErrInvalidInput
	}
}

func isClientVisibleResponseStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "submitted", "accepted":
		return true
	default:
		return false
	}
}

func isCourseOwner(target courseAttachmentTarget, callerID string) bool {
	return (target.CoachID != nil && *target.CoachID == callerID) || (target.CreatedBy != nil && *target.CreatedBy == callerID)
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
