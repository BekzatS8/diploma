package responses

import (
	"context"
	"errors"
	"strings"

	ratings "buhpro/internal/modules/ratingsanctions"
	"buhpro/internal/platform/payments"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound      = errors.New("response not found")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidRole   = errors.New("invalid role")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInvalidStatus = errors.New("invalid status")
	ErrDuplicate     = errors.New("duplicate response")
)

type Service struct {
	repo                *Repository
	paymentProvider     payments.Provider
	paymentProviderName string
	submissionFee       float64
	defaultCurrency     string
	ratingService       *ratings.Service
}

func NewService(repo *Repository, paymentProvider payments.Provider, paymentProviderName string, submissionFee float64, defaultCurrency string, ratingService *ratings.Service) *Service {
	return &Service{repo: repo, paymentProvider: paymentProvider, paymentProviderName: paymentProviderName, submissionFee: submissionFee, defaultCurrency: strings.ToUpper(strings.TrimSpace(defaultCurrency)), ratingService: ratingService}
}

func (s *Service) Create(ctx context.Context, orderID, userID, role string, req CreateResponseRequest) (Response, error) {
	if role != "executor" {
		return Response{}, ErrInvalidRole
	}
	if blocked, err := s.ratingService.HasActiveResponseRestriction(ctx, userID); err != nil {
		return Response{}, err
	} else if blocked {
		return Response{}, ErrForbidden
	}
	clientID, orderStatus, deleted, err := s.repo.GetOrderForResponse(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, ErrNotFound
		}
		return Response{}, err
	}
	if deleted || orderStatus != "published" {
		return Response{}, ErrInvalidStatus
	}
	if clientID == userID {
		return Response{}, ErrForbidden
	}

	currency := s.defaultCurrency
	if req.Currency != nil && strings.TrimSpace(*req.Currency) != "" {
		currency = strings.ToUpper(strings.TrimSpace(*req.Currency))
		if len(currency) != 3 {
			return Response{}, ErrInvalidInput
		}
	}
	if req.ProposedAmount != nil && *req.ProposedAmount <= 0 {
		return Response{}, ErrInvalidInput
	}
	if req.CoverLetter != nil {
		v := strings.TrimSpace(*req.CoverLetter)
		if len(v) > 5000 {
			return Response{}, ErrInvalidInput
		}
		req.CoverLetter = &v
	}

	res, err := s.repo.Create(ctx, CreateParams{OrderID: orderID, ExecutorID: userID, CoverLetter: req.CoverLetter, ProposedAmount: req.ProposedAmount, Currency: currency})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Response{}, ErrDuplicate
		}
		return Response{}, err
	}
	res.OrderClientID = clientID
	res.OrderStatus = orderStatus
	return res, nil
}

func (s *Service) UpdateDraft(ctx context.Context, orderID, responseID, userID, role string, req UpdateResponseRequest) (Response, error) {
	if role != "executor" {
		return Response{}, ErrInvalidRole
	}
	current, err := s.repo.GetMyByID(ctx, orderID, responseID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, ErrNotFound
		}
		return Response{}, err
	}
	if current.Status != StatusDraft {
		return Response{}, ErrInvalidStatus
	}
	if req.ProposedAmount != nil && *req.ProposedAmount <= 0 {
		return Response{}, ErrInvalidInput
	}
	if req.CoverLetter != nil {
		v := strings.TrimSpace(*req.CoverLetter)
		if len(v) > 5000 {
			return Response{}, ErrInvalidInput
		}
		req.CoverLetter = &v
	}
	if req.Currency != nil {
		v := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if len(v) != 3 {
			return Response{}, ErrInvalidInput
		}
		req.Currency = &v
	}
	updated, err := s.repo.UpdateDraft(ctx, orderID, responseID, userID, UpdateParams{CoverLetter: req.CoverLetter, ProposedAmount: req.ProposedAmount, Currency: req.Currency})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, ErrNotFound
		}
		return Response{}, err
	}
	updated.OrderTitle = current.OrderTitle
	return updated, nil
}

func (s *Service) Submit(ctx context.Context, orderID, responseID, userID, role string) (Response, PaymentTransaction, payments.ChargeResponse, error) {
	if role != "executor" {
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidRole
	}
	if blocked, err := s.ratingService.HasActiveResponseRestriction(ctx, userID); err != nil {
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	} else if blocked {
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrForbidden
	}
	current, err := s.repo.GetMyByID(ctx, orderID, responseID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrNotFound
		}
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	}
	if current.Status != StatusDraft || current.OrderStatus != "published" {
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidStatus
	}
	if current.CoverLetter == nil || strings.TrimSpace(*current.CoverLetter) == "" {
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidInput
	}

	charge, err := s.paymentProvider.CreateCharge(ctx, payments.ChargeRequest{OrderID: current.ID, AmountCents: int64(s.submissionFee * 100), CurrencyCode: s.defaultCurrency, Description: "Response submission fee"})
	if err != nil {
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	}
	updated, pay, err := s.repo.SubmitWithPayment(ctx, orderID, responseID, userID, s.submissionFee, s.defaultCurrency, s.paymentProviderName, charge.TransactionID, charge.RedirectURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrNotFound
		}
		return Response{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	}
	updated.OrderTitle = current.OrderTitle
	return updated, pay, charge, nil
}

func (s *Service) Cancel(ctx context.Context, orderID, responseID, userID, role string) (Response, error) {
	if role != "executor" {
		return Response{}, ErrInvalidRole
	}
	updated, err := s.repo.Cancel(ctx, orderID, responseID, userID, "cancelled by executor")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, ErrNotFound
		}
		return Response{}, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, orderID, responseID, userID, role string) error {
	if role != "executor" {
		return ErrInvalidRole
	}
	current, err := s.repo.GetMyByID(ctx, orderID, responseID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !CanDelete(current.Status) {
		return ErrInvalidStatus
	}
	return s.repo.SoftDelete(ctx, orderID, responseID, userID)
}

func (s *Service) ListOrderMy(ctx context.Context, orderID, userID, role string, q ListQuery) ([]Response, int64, error) {
	if role != "executor" {
		return nil, 0, ErrInvalidRole
	}
	return s.repo.ListExecutorByOrder(ctx, orderID, userID, q)
}

func (s *Service) GetOrderMyByID(ctx context.Context, orderID, responseID, userID, role string) (Response, error) {
	if role != "executor" {
		return Response{}, ErrInvalidRole
	}
	item, err := s.repo.GetMyByID(ctx, orderID, responseID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, ErrNotFound
		}
		return Response{}, err
	}
	return item, nil
}

func (s *Service) ListMy(ctx context.Context, userID, role string, q ListQuery) ([]Response, int64, error) {
	if role == "admin" {
		return s.repo.ListAll(ctx, q)
	}
	if role != "executor" {
		return nil, 0, ErrInvalidRole
	}
	return s.repo.ListExecutor(ctx, userID, q)
}

func (s *Service) GetMyByID(ctx context.Context, responseID, userID, role string) (Response, error) {
	if role == "admin" {
		item, err := s.repo.GetByID(ctx, responseID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Response{}, ErrNotFound
			}
			return Response{}, err
		}
		return item, nil
	}
	if role != "executor" {
		return Response{}, ErrInvalidRole
	}
	item, err := s.repo.GetExecutorByID(ctx, responseID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, ErrNotFound
		}
		return Response{}, err
	}
	return item, nil
}

func (s *Service) ListClientOrder(ctx context.Context, orderID, userID, role string, q ListQuery) ([]Response, int64, error) {
	if role != "client" && role != "admin" {
		return nil, 0, ErrInvalidRole
	}
	if role == "client" {
		clientID, _, deleted, err := s.repo.GetOrderForResponse(ctx, orderID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, ErrNotFound
			}
			return nil, 0, err
		}
		if deleted || clientID != userID {
			return nil, 0, ErrForbidden
		}
	}
	return s.repo.ListForClientOrder(ctx, orderID, q)
}

func (s *Service) GetClientOrderByID(ctx context.Context, orderID, responseID, userID, role string) (Response, error) {
	if role != "client" && role != "admin" {
		return Response{}, ErrInvalidRole
	}
	if role == "client" {
		clientID, _, deleted, err := s.repo.GetOrderForResponse(ctx, orderID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Response{}, ErrNotFound
			}
			return Response{}, err
		}
		if deleted || clientID != userID {
			return Response{}, ErrForbidden
		}
	}
	item, err := s.repo.GetClientOrderResponse(ctx, orderID, responseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, ErrNotFound
		}
		return Response{}, err
	}
	return item, nil
}
