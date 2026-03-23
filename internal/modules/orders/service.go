package orders

import (
	"context"
	"errors"
	"strings"

	"buhpro/internal/platform/payments"

	"github.com/jackc/pgx/v5"
)

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrForbidden               = errors.New("forbidden")
	ErrInvalidRole             = errors.New("invalid role")
	ErrInvalidInput            = errors.New("invalid input")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

type Service struct {
	repo                *Repository
	paymentProvider     payments.Provider
	paymentProviderName string
	postingFee          float64
	defaultCurrency     string
}

func NewService(repo *Repository, paymentProvider payments.Provider, paymentProviderName string, postingFee float64, defaultCurrency string) *Service {
	return &Service{
		repo:                repo,
		paymentProvider:     paymentProvider,
		paymentProviderName: paymentProviderName,
		postingFee:          postingFee,
		defaultCurrency:     strings.ToUpper(strings.TrimSpace(defaultCurrency)),
	}
}

func (s *Service) Create(ctx context.Context, userID, role string, req CreateOrderRequest) (Order, error) {
	if role != "client" {
		return Order{}, ErrInvalidRole
	}
	catID, err := s.resolveCategory(ctx, req.CategoryID, req.CategorySlug)
	if err != nil {
		return Order{}, err
	}
	currency := s.defaultCurrency
	if req.Currency != nil && strings.TrimSpace(*req.Currency) != "" {
		currency = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}

	order, err := s.repo.Create(ctx, CreateOrderParams{
		ClientID:     userID,
		CategoryID:   catID,
		Title:        strings.TrimSpace(req.Title),
		Description:  strings.TrimSpace(req.Description),
		BudgetAmount: req.BudgetAmount,
		Currency:     currency,
	})
	if err != nil {
		return Order{}, err
	}
	return s.enrichCategory(ctx, order)
}

func (s *Service) ListMy(ctx context.Context, userID, role string, q MyOrdersQuery) ([]Order, int64, error) {
	if role != "client" {
		return nil, 0, ErrInvalidRole
	}
	return s.repo.ListMy(ctx, userID, q)
}

func (s *Service) GetMyByID(ctx context.Context, userID, role, id string) (Order, *PaymentTransaction, error) {
	if role != "client" {
		return Order{}, nil, ErrInvalidRole
	}
	order, err := s.repo.GetMyByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, nil, ErrOrderNotFound
		}
		return Order{}, nil, err
	}
	payment, err := s.repo.LatestPaymentByOrderID(ctx, order.ID)
	if err != nil {
		return Order{}, nil, err
	}
	return order, payment, nil
}

func (s *Service) UpdateDraft(ctx context.Context, userID, role, id string, req UpdateOrderRequest) (Order, error) {
	if role != "client" {
		return Order{}, ErrInvalidRole
	}
	current, err := s.repo.GetMyByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, err
	}
	if current.Status != "draft" {
		return Order{}, ErrInvalidStatusTransition
	}

	catID, err := s.resolveCategory(ctx, req.CategoryID, req.CategorySlug)
	if err != nil {
		return Order{}, err
	}
	if req.BudgetAmount != nil && *req.BudgetAmount <= 0 {
		return Order{}, ErrInvalidInput
	}

	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		req.Title = &v
		if v == "" {
			return Order{}, ErrInvalidInput
		}
	}
	if req.Description != nil {
		v := strings.TrimSpace(*req.Description)
		req.Description = &v
		if v == "" {
			return Order{}, ErrInvalidInput
		}
	}
	if req.Currency != nil {
		v := strings.ToUpper(strings.TrimSpace(*req.Currency))
		req.Currency = &v
		if len(v) != 3 {
			return Order{}, ErrInvalidInput
		}
	}

	updated, err := s.repo.UpdateDraft(ctx, id, userID, UpdateOrderParams{
		Title:        req.Title,
		Description:  req.Description,
		CategoryID:   catID,
		BudgetAmount: req.BudgetAmount,
		Currency:     req.Currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, err
	}
	return s.enrichCategory(ctx, updated)
}

func (s *Service) DeleteMy(ctx context.Context, userID, role, id string) error {
	if role != "client" {
		return ErrInvalidRole
	}
	order, err := s.repo.GetMyByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}
		return err
	}
	if order.Status != "draft" && order.Status != "cancelled" {
		return ErrInvalidStatusTransition
	}
	return s.repo.SoftDelete(ctx, id, userID)
}

func (s *Service) Submit(ctx context.Context, userID, role, id string) (Order, PaymentTransaction, payments.ChargeResponse, error) {
	if role != "client" {
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidRole
	}
	order, err := s.repo.GetMyByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrOrderNotFound
		}
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	}
	if order.Status != "draft" {
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidStatusTransition
	}
	if strings.TrimSpace(order.Title) == "" || strings.TrimSpace(order.Description) == "" || order.BudgetAmount <= 0 || order.CategoryID == nil {
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidInput
	}

	amountCents := int64(s.postingFee * 100)
	charge, err := s.paymentProvider.CreateCharge(ctx, payments.ChargeRequest{
		OrderID:      order.ID,
		AmountCents:  amountCents,
		CurrencyCode: s.defaultCurrency,
		Description:  "Order posting fee",
	})
	if err != nil {
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	}

	updated, tx, err := s.repo.SubmitWithPayment(ctx, id, userID, s.postingFee, s.defaultCurrency, s.paymentProviderName, charge.TransactionID, charge.RedirectURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrOrderNotFound
		}
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	}
	return updated, tx, charge, nil
}

func (s *Service) Cancel(ctx context.Context, userID, role, id string) (Order, error) {
	if role != "client" {
		return Order{}, ErrInvalidRole
	}
	updated, err := s.repo.Cancel(ctx, id, userID, "cancelled by client")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, err
	}
	return s.enrichCategory(ctx, updated)
}

func (s *Service) ListPublic(ctx context.Context, q PublicOrdersQuery) ([]Order, int64, error) {
	return s.repo.ListPublic(ctx, q)
}

func (s *Service) GetByID(ctx context.Context, id string, userID string, role string) (Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, err
	}
	if order.DeletedAt != nil {
		return Order{}, ErrOrderNotFound
	}
	if order.Status == "published" {
		return order, nil
	}
	if role == "admin" {
		return order, nil
	}
	if role == "client" && userID != "" && order.ClientID == userID {
		return order, nil
	}
	return Order{}, ErrForbidden
}

func (s *Service) resolveCategory(ctx context.Context, categoryID *int64, slug *string) (*int64, error) {
	if slug == nil || strings.TrimSpace(*slug) == "" {
		return categoryID, nil
	}
	resolved, err := s.repo.ResolveCategoryIDBySlug(ctx, strings.TrimSpace(*slug))
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, ErrInvalidInput
	}
	return resolved, nil
}

func (s *Service) enrichCategory(ctx context.Context, o Order) (Order, error) {
	if o.CategoryID == nil {
		return o, nil
	}
	full, err := s.repo.GetByID(ctx, o.ID)
	if err != nil {
		return o, nil
	}
	return full, nil
}
