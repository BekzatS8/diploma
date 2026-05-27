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
	ErrInvalidState  = errors.New("invalid state")
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
	preconditions, can, err := s.repo.CanCreateReview(ctx, orderID, userID, role)
	if err != nil {
		return Review{}, err
	}
	if !can {
		return Review{}, ErrInvalidState
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Review{}, err
	}
	defer tx.Rollback(ctx)

	item, err := s.repo.CreateTx(ctx, tx, CreateReviewParams{
		OrderID:      orderID,
		ClientID:     preconditions.ClientID,
		ExecutorID:   preconditions.ExecutorID,
		ReviewerID:   preconditions.ReviewerID,
		RevieweeID:   preconditions.RevieweeID,
		RevieweeRole: preconditions.RevieweeRole,
		Direction:    preconditions.Direction,
		Rating:       rating,
		Comment:      comment,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Review{}, ErrAlreadyExists
		}
		return Review{}, err
	}
	ratingResult := ratings.RecalculateResult{}
	if preconditions.RevieweeRole == "executor" {
		ratingResult, err = s.ratingService.RecalculateAndApplyTx(ctx, tx, preconditions.RevieweeID)
		if err != nil {
			return Review{}, err
		}
	} else if err := s.repo.RecalculateClientRatingTx(ctx, tx, preconditions.RevieweeID); err != nil {
		return Review{}, err
	}
	if _, err := s.repo.CreateEntityTx(ctx, tx, CreateEntityReviewParams{
		AuthorID:   preconditions.ReviewerID,
		TargetType: "order",
		TargetID:   orderID,
		Rating:     rating,
		Comment:    comment,
		Metadata: map[string]any{
			"legacy_review_id": item.ID,
			"reviewer_id":      preconditions.ReviewerID,
			"reviewee_id":      preconditions.RevieweeID,
			"direction":        preconditions.Direction,
		},
	}); err != nil {
		return Review{}, err
	}
	if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "order", orderID); err != nil {
		return Review{}, err
	}
	if _, err := s.repo.CreateEntityTx(ctx, tx, CreateEntityReviewParams{
		AuthorID:   preconditions.ReviewerID,
		TargetType: "user",
		TargetID:   preconditions.RevieweeID,
		Rating:     rating,
		Comment:    comment,
		Metadata: map[string]any{
			"legacy_review_id": item.ID,
			"order_id":         orderID,
			"direction":        preconditions.Direction,
		},
	}); err != nil {
		return Review{}, err
	}
	if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "user", preconditions.RevieweeID); err != nil {
		return Review{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Review{}, err
	}
	if s.notifier != nil {
		_, _ = s.notifier.EmitInApp(ctx, preconditions.RevieweeID, notifications.TypeReviewCreated, map[string]any{
			"order_id":  orderID,
			"review_id": item.ID,
			"rating":    item.Rating,
			"direction": preconditions.Direction,
		})
		if ratingResult.SanctionCreated {
			_, _ = s.notifier.EmitInApp(ctx, preconditions.RevieweeID, notifications.TypeSanctionCreated, map[string]any{
				"sanction_id": ratingResult.SanctionID,
				"reason":      ratingResult.SanctionReason,
				"order_id":    orderID,
				"review_id":   item.ID,
			})
		}
		if ratingResult.AutoCourseAssigned {
			_, _ = s.notifier.EmitInApp(ctx, preconditions.RevieweeID, notifications.TypeCourseAssigned, map[string]any{
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

func (s *Service) ListUser(ctx context.Context, userID string, page, size int) ([]Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListUser(ctx, userID, ListQuery{Page: page, PageSize: size})
}

func (s *Service) ListAuthored(ctx context.Context, userID string, page, size int) ([]MyReview, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListAuthored(ctx, userID, ListQuery{Page: page, PageSize: size})
}

func (s *Service) GetAuthored(ctx context.Context, id, userID string) (MyReview, error) {
	item, err := s.repo.GetAuthored(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyReview{}, ErrNotFound
		}
		return MyReview{}, err
	}
	return item, nil
}

func (s *Service) UpdateAuthored(ctx context.Context, id, userID string, rating int, comment *string) (MyReview, error) {
	if rating < 1 || rating > 5 {
		return MyReview{}, ErrInvalidInput
	}
	if comment != nil {
		v := strings.TrimSpace(*comment)
		comment = &v
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MyReview{}, err
	}
	defer tx.Rollback(ctx)

	orderReview, err := s.repo.UpdateOrderReviewOwnedTx(ctx, tx, id, userID, rating, comment)
	if err == nil {
		if err := s.repo.UpdateMirroredOrderEntityReviewsTx(ctx, tx, orderReview.ID, rating, comment); err != nil {
			return MyReview{}, err
		}
		if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "order", orderReview.OrderID); err != nil {
			return MyReview{}, err
		}
		if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "user", orderReview.RevieweeID); err != nil {
			return MyReview{}, err
		}
		if err := s.recalculateReviewedUserTx(ctx, tx, orderReview.RevieweeID, orderReview.RevieweeRole); err != nil {
			return MyReview{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return MyReview{}, err
		}
		return reviewToMyReview(orderReview), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return MyReview{}, err
	}

	entityReview, err := s.repo.UpdateEntityReviewOwnedTx(ctx, tx, id, userID, rating, comment)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyReview{}, ErrNotFound
		}
		return MyReview{}, err
	}
	if err := s.repo.RecalculateEntityRatingTx(ctx, tx, entityReview.TargetType, entityReview.TargetID); err != nil {
		return MyReview{}, err
	}
	if entityReview.TargetType == "course" {
		if ownerID, updated, err := s.repo.UpdateCourseOwnerMirrorTx(ctx, tx, entityReview.ID, rating, comment); err != nil {
			return MyReview{}, err
		} else if updated {
			if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "user", ownerID); err != nil {
				return MyReview{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return MyReview{}, err
	}
	return entityToMyReview(entityReview), nil
}

func (s *Service) DeleteAuthored(ctx context.Context, id, userID string) (MyReview, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MyReview{}, err
	}
	defer tx.Rollback(ctx)

	orderReview, err := s.repo.DeleteOrderReviewOwnedTx(ctx, tx, id, userID)
	if err == nil {
		if err := s.repo.DeleteMirroredOrderEntityReviewsTx(ctx, tx, orderReview.ID); err != nil {
			return MyReview{}, err
		}
		if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "order", orderReview.OrderID); err != nil {
			return MyReview{}, err
		}
		if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "user", orderReview.RevieweeID); err != nil {
			return MyReview{}, err
		}
		if err := s.recalculateReviewedUserTx(ctx, tx, orderReview.RevieweeID, orderReview.RevieweeRole); err != nil {
			return MyReview{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return MyReview{}, err
		}
		return reviewToMyReview(orderReview), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return MyReview{}, err
	}

	entityReview, err := s.repo.DeleteEntityReviewOwnedTx(ctx, tx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyReview{}, ErrNotFound
		}
		return MyReview{}, err
	}
	if err := s.repo.RecalculateEntityRatingTx(ctx, tx, entityReview.TargetType, entityReview.TargetID); err != nil {
		return MyReview{}, err
	}
	if entityReview.TargetType == "course" {
		if ownerID, deleted, err := s.repo.DeleteCourseOwnerMirrorTx(ctx, tx, entityReview.ID); err != nil {
			return MyReview{}, err
		} else if deleted {
			if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "user", ownerID); err != nil {
				return MyReview{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return MyReview{}, err
	}
	return entityToMyReview(entityReview), nil
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
	if targetType != "course" {
		return EntityReview{}, ErrForbidden
	}
	canReviewCourse, err := s.repo.CanCreateCourseReview(ctx, userID, role, targetID)
	if err != nil {
		return EntityReview{}, err
	}
	if !canReviewCourse {
		return EntityReview{}, ErrForbidden
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
	if courseOwnerID, ok, err := s.repo.GetCourseOwner(ctx, targetID); err != nil {
		return EntityReview{}, err
	} else if ok && courseOwnerID != userID {
		if _, err := s.repo.CreateEntityTx(ctx, tx, CreateEntityReviewParams{
			AuthorID:   userID,
			TargetType: "user",
			TargetID:   courseOwnerID,
			Rating:     rating,
			Comment:    comment,
			Metadata: map[string]any{
				"source":           "course_review",
				"course_id":        targetID,
				"course_review_id": item.ID,
			},
		}); err != nil {
			return EntityReview{}, err
		}
		if err := s.repo.RecalculateEntityRatingTx(ctx, tx, "user", courseOwnerID); err != nil {
			return EntityReview{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return EntityReview{}, err
	}
	return item, nil
}

func (s *Service) recalculateReviewedUserTx(ctx context.Context, tx pgx.Tx, revieweeID, revieweeRole string) error {
	if revieweeRole == "executor" {
		_, err := s.ratingService.RecalculateAndApplyTx(ctx, tx, revieweeID)
		return err
	}
	if revieweeRole == "client" {
		return s.repo.RecalculateClientRatingTx(ctx, tx, revieweeID)
	}
	return nil
}

func reviewToMyReview(item Review) MyReview {
	orderID := item.OrderID
	targetID := item.RevieweeID
	revieweeID := item.RevieweeID
	revieweeRole := item.RevieweeRole
	direction := item.Direction
	return MyReview{
		ID:           item.ID,
		Source:       "order",
		OrderID:      &orderID,
		TargetID:     &targetID,
		RevieweeID:   &revieweeID,
		RevieweeRole: &revieweeRole,
		Direction:    &direction,
		Rating:       item.Rating,
		Comment:      item.Comment,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func entityToMyReview(item EntityReview) MyReview {
	targetType := item.TargetType
	targetID := item.TargetID
	return MyReview{
		ID:         item.ID,
		Source:     "entity",
		TargetType: &targetType,
		TargetID:   &targetID,
		Rating:     item.Rating,
		Comment:    item.Comment,
		Metadata:   item.Metadata,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
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
