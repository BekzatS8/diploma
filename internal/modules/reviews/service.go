package reviews

import (
	"context"
	"errors"
	"strings"

	ratings "buhpro/internal/modules/ratingsanctions"

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
}

func NewService(repo *Repository, db *pgxpool.Pool, ratingService *ratings.Service) *Service {
	return &Service{repo: repo, db: db, ratingService: ratingService}
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
	if err := s.ratingService.RecalculateAndApplyTx(ctx, tx, executorID); err != nil {
		return Review{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Review{}, err
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
