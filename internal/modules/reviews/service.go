package reviews

import (
	"context"
	"errors"
	"strings"

	notifications "buhpro/internal/modules/notifications"
	ratings "buhpro/internal/modules/ratingsanctions"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrForbidden     = errors.New("forbidden")
	ErrNotFound      = errors.New("not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrAlreadyExists = errors.New("already exists")
)

type Service struct {
	repo          *Repository
	db            *pgxpool.Pool
	ratingService *ratings.Service
	notifier      *notifications.Service
}

func NewService(repo *Repository, db *pgxpool.Pool, ratingService *ratings.Service, notifier *notifications.Service) *Service {
	return &Service{repo: repo, db: db, ratingService: ratingService, notifier: notifier}
}

func (s *Service) Create(ctx context.Context, orderID, userID, role string, rating int, comment *string) (Review, error) {
	ok, err := s.repo.IsOwnerOrAdmin(ctx, orderID, userID, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Review{}, ErrNotFound
		}
		return Review{}, err
	}
	if !ok {
		return Review{}, ErrForbidden
	}
	if rating < 1 || rating > 5 {
		return Review{}, ErrInvalidInput
	}
	if comment != nil {
		v := strings.TrimSpace(*comment)
		comment = &v
	}
	clientID, executorID, can, err := s.repo.CanCreateReview(ctx, orderID)
	if err != nil {
		return Review{}, err
	}
	if !can {
		return Review{}, ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Review{}, err
	}
	defer tx.Rollback(ctx)

	item, err := s.repo.CreateTx(ctx, tx, CreateReviewParams{OrderID: orderID, ClientID: clientID, ExecutorID: executorID, Rating: rating, Comment: comment})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Review{}, ErrAlreadyExists
		}
		return Review{}, err
	}
	ratingResult, err := s.ratingService.RecalculateAndApplyTx(ctx, tx, executorID)
	if err != nil {
		return Review{}, err
	}
	if _, err := s.repo.CreateEntityTx(ctx, tx, CreateEntityReviewParams{
		AuthorID:   clientID,
		TargetType: "order",
		TargetID:   orderID,
		Rating:     rating,
		Comment:    comment,
		Metadata: map[string]any{
			"legacy_review_id": item.ID,
			"executor_id":      executorID,
		},
	}); err != nil {
		return Review{}, err
	}
	if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "order", orderID); err != nil {
		return Review{}, err
	}
	if _, err := s.repo.CreateEntityTx(ctx, tx, CreateEntityReviewParams{
		AuthorID:   clientID,
		TargetType: "user",
		TargetID:   executorID,
		Rating:     rating,
		Comment:    comment,
		Metadata: map[string]any{
			"legacy_review_id": item.ID,
			"order_id":         orderID,
		},
	}); err != nil {
		return Review{}, err
	}
	if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "user", executorID); err != nil {
		return Review{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Review{}, err
	}
	if s.notifier != nil {
		_, _ = s.notifier.EmitInApp(ctx, executorID, notifications.TypeReviewCreated, map[string]any{
			"order_id":  orderID,
			"review_id": item.ID,
			"rating":    item.Rating,
		})
		if ratingResult.SanctionCreated {
			_, _ = s.notifier.EmitInApp(ctx, executorID, notifications.TypeSanctionCreated, map[string]any{
				"sanction_id": ratingResult.SanctionID,
				"reason":      ratingResult.SanctionReason,
				"order_id":    orderID,
				"review_id":   item.ID,
			})
		}
		if ratingResult.AutoCourseAssigned {
			_, _ = s.notifier.EmitInApp(ctx, executorID, notifications.TypeCourseAssigned, map[string]any{
				"course_id":            ratingResult.AutoCourseID,
				"course_assignment_id": ratingResult.AutoCourseAssignmentID,
				"sanction_id":          ratingResult.SanctionID,
				"source":               "sanction_auto",
			})
		}
	}
	return item, nil
}

func (s *Service) GetByOrder(ctx context.Context, orderID, userID, role string) (Review, error) {
	ok, err := s.repo.IsOwnerOrAdmin(ctx, orderID, userID, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Review{}, ErrNotFound
		}
		return Review{}, err
	}
	if !ok {
		return Review{}, ErrForbidden
	}
	item, err := s.repo.GetByOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Review{}, ErrNotFound
		}
		return Review{}, err
	}
	return item, nil
}

func (s *Service) ListExecutor(ctx context.Context, executorID string, page, size int) ([]Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListExecutor(ctx, executorID, ListQuery{Page: page, PageSize: size})
}

func (s *Service) CreateEntity(ctx context.Context, userID, role, targetType, targetID string, rating int, comment *string, metadata map[string]any) (EntityReview, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(role) == "" {
		return EntityReview{}, ErrForbidden
	}
	targetType = normalizeTargetType(targetType)
	if !isValidTargetType(targetType) {
		return EntityReview{}, ErrInvalidInput
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return EntityReview{}, ErrInvalidInput
	}
	if rating < 1 || rating > 5 {
		return EntityReview{}, ErrInvalidInput
	}
	if comment != nil {
		v := strings.TrimSpace(*comment)
		comment = &v
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return EntityReview{}, err
	}
	defer tx.Rollback(ctx)

	item, err := s.repo.CreateEntityTx(ctx, tx, CreateEntityReviewParams{
		AuthorID:   userID,
		TargetType: targetType,
		TargetID:   targetID,
		Rating:     rating,
		Comment:    comment,
		Metadata:   metadata,
	})
	if err != nil {
		return EntityReview{}, err
	}
	if err := s.repo.RecalculateEntityRatingTx(ctx, tx, targetType, targetID); err != nil {
		return EntityReview{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EntityReview{}, err
	}
	return item, nil
}

func (s *Service) ListByTarget(ctx context.Context, targetType, targetID string, page, size int) ([]EntityReview, int64, error) {
	targetType = normalizeTargetType(targetType)
	if !isValidTargetType(targetType) {
		return nil, 0, ErrInvalidInput
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return nil, 0, ErrInvalidInput
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListByTarget(ctx, targetType, targetID, ListQuery{Page: page, PageSize: size})
}

func (s *Service) GetRatingSummary(ctx context.Context, targetType, targetID string) (RatingSummary, error) {
	targetType = normalizeTargetType(targetType)
	if !isValidTargetType(targetType) {
		return RatingSummary{}, ErrInvalidInput
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return RatingSummary{}, ErrInvalidInput
	}
	return s.repo.GetRatingSummary(ctx, targetType, targetID)
}

func normalizeTargetType(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func isValidTargetType(value string) bool {
	switch value {
	case "user", "client", "executor", "coach", "profile", "order", "response", "review", "course", "course_material":
		return true
	default:
		return false
	}
}
