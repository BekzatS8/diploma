package orders

import (
	"context"
	"errors"
	"math"
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
	ErrInsufficientBalance     = errors.New("insufficient balance")
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
		if len(currency) != 3 {
			return Order{}, ErrInvalidInput
		}
	}
	region := normalizeString(req.Region)
	promotions := normalizePromotions(req.Promotions)

	order, err := s.repo.Create(ctx, CreateOrderParams{
		ClientID:     userID,
		CategoryID:   catID,
		Title:        strings.TrimSpace(req.Title),
		Description:  strings.TrimSpace(req.Description),
		BudgetAmount: req.BudgetAmount,
		Currency:     currency,
		DeadlineAt:   req.DeadlineAt,
		Region:       region,
		Promotions:   promotions,
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
	if current.Status != StatusDraft {
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
	req.Region = normalizeString(req.Region)
	req.Promotions = normalizePromotions(req.Promotions)

	updated, err := s.repo.UpdateDraft(ctx, id, userID, UpdateOrderParams{
		Title:        req.Title,
		Description:  req.Description,
		CategoryID:   catID,
		BudgetAmount: req.BudgetAmount,
		Currency:     req.Currency,
		DeadlineAt:   req.DeadlineAt,
		Region:       req.Region,
		Promotions:   req.Promotions,
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
	if !CanDelete(order.Status) {
		return ErrInvalidStatusTransition
	}
	return s.repo.SoftDelete(ctx, id, userID)
}

func (s *Service) Submit(ctx context.Context, userID, role, id string) (Order, PaymentTransaction, payments.ChargeResponse, error) {
	return s.submit(ctx, userID, role, id, nil)
}

func (s *Service) SubmitExpectedAmount(ctx context.Context, userID, role, id string, expectedAmount float64) (Order, PaymentTransaction, payments.ChargeResponse, error) {
	return s.submit(ctx, userID, role, id, &expectedAmount)
}

func (s *Service) submit(ctx context.Context, userID, role, id string, expectedAmount *float64) (Order, PaymentTransaction, payments.ChargeResponse, error) {
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
	if order.Status != StatusDraft && order.Status != StatusPaymentPending {
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidStatusTransition
	}
	if strings.TrimSpace(order.Title) == "" || strings.TrimSpace(order.Description) == "" || order.BudgetAmount <= 0 || order.CategoryID == nil {
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrInvalidInput
	}

	promotionFee := promotionFee(order.PromotionOptions)
	escrowAmount := order.BudgetAmount
	totalCharge := s.postingFee + promotionFee + escrowAmount

	updated, err := s.repo.PublishFromDraft(ctx, id, userID, s.postingFee, promotionFee, escrowAmount, totalCharge)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, ErrOrderNotFound
		}
		return Order{}, PaymentTransaction{}, payments.ChargeResponse{}, err
	}
	return updated, PaymentTransaction{}, payments.ChargeResponse{Status: "succeeded"}, nil
}

func amountToCents(value float64) int64 {
	return int64(math.Round(value * 100))
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
	if order.Status == StatusPublished {
		return order, nil
	}
	if role == "admin" {
		return order, nil
	}
	if role == "executor" && userID != "" && order.SelectedExecutorID != nil && *order.SelectedExecutorID == userID {
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

func normalizeString(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizePromotions(items []string) []string {
	if items == nil {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		key := strings.TrimSpace(strings.ToLower(item))
		switch key {
		case "top", "promotion_top", "raise_top":
			key = "top"
		case "pin", "pinned", "promotion_pin":
			key = "pin"
		case "highlight", "highlighted", "promotion_highlight":
			key = "highlight"
		default:
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func promotionFee(items []string) float64 {
	total := 0.0
	for _, item := range items {
		switch item {
		case "top":
			total += 1000
		case "pin":
			total += 1500
		case "highlight":
			total += 500
		}
	}
	return total
}
