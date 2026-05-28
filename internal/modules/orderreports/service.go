package orderreports

import (
	"context"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, orderID, reporterID, reason string) (CreateReportResponse, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 10 {
		return CreateReportResponse{}, ErrInvalidInput
	}
	ok, err := s.repo.OrderReportable(ctx, orderID)
	if err != nil {
		return CreateReportResponse{}, err
	}
	if !ok {
		return CreateReportResponse{}, ErrOrderNotReportable
	}
	id, err := s.repo.Create(ctx, orderID, reporterID, reason)
	if err != nil {
		return CreateReportResponse{}, err
	}
	return CreateReportResponse{
		ID:      id,
		Status:  string(StatusPending),
		Message: "Жалоба отправлена администратору",
	}, nil
}

func (s *Service) List(ctx context.Context, status string, page, pageSize int) (ListResponse, error) {
	st := Status(strings.TrimSpace(status))
	if st != "" && st != StatusPending && st != StatusDismissed && st != StatusOrderRemoved {
		return ListResponse{}, ErrInvalidInput
	}
	items, total, err := s.repo.List(ctx, st, page, pageSize)
	if err != nil {
		return ListResponse{}, err
	}
	views := make([]ReportView, 0, len(items))
	for _, item := range items {
		views = append(views, toView(item))
	}
	return ListResponse{
		Items:    views,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (s *Service) Dismiss(ctx context.Context, reportID, adminID, notes string) (ReviewResponse, error) {
	if err := s.repo.Review(ctx, reportID, adminID, StatusDismissed, notes); err != nil {
		return ReviewResponse{}, err
	}
	return ReviewResponse{
		ID:      reportID,
		Status:  string(StatusDismissed),
		Message: "Жалоба отклонена, заказ оставлен на платформе",
	}, nil
}

func (s *Service) RemoveOrder(ctx context.Context, reportID, adminID, notes string) (ReviewResponse, error) {
	report, err := s.repo.GetByID(ctx, reportID)
	if err != nil {
		return ReviewResponse{}, err
	}
	if report.Status != StatusPending {
		return ReviewResponse{}, ErrAlreadyReviewed
	}
	if err := s.repo.AdminSoftDeleteOrder(ctx, report.OrderID); err != nil {
		return ReviewResponse{}, err
	}
	if err := s.repo.Review(ctx, reportID, adminID, StatusOrderRemoved, notes); err != nil {
		return ReviewResponse{}, err
	}
	return ReviewResponse{
		ID:      reportID,
		Status:  string(StatusOrderRemoved),
		Message: "Заказ снят с платформы",
	}, nil
}

func toView(item Report) ReportView {
	return ReportView{
		ID:               item.ID,
		OrderID:          item.OrderID,
		ReporterID:       item.ReporterID,
		ReporterEmail:    item.ReporterEmail,
		ReporterName:     item.ReporterName,
		Reason:           item.Reason,
		Status:           string(item.Status),
		AdminNotes:       item.AdminNotes,
		OrderTitle:       item.OrderTitle,
		OrderDescription: item.OrderDesc,
		OrderBudget:      item.OrderBudget,
		OrderCurrency:    item.OrderCurrency,
		OrderStatus:      item.OrderStatus,
		CreatedAt:        item.CreatedAt,
		ReviewedAt:       item.ReviewedAt,
	}
}
