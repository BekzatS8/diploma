package orders

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	order, err := h.service.Create(c.Request.Context(), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toOrderResponse(order, true))
}

func (h *Handler) ListMy(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	query := MyOrdersQuery{
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     parseIntDefault(c.Query("page"), 1),
		PageSize: parseIntDefault(c.Query("page_size"), 20),
	}
	if query.Page < 1 || query.PageSize < 1 || query.PageSize > 100 {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid pagination")
		return
	}
	items, total, err := h.service.ListMy(c.Request.Context(), user.UserID, user.PrimaryRole(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, OrdersListResponse{Items: toOrderResponses(items, true), Page: query.Page, PageSize: query.PageSize, Total: total})
}

func (h *Handler) GetMyByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	order, payment, err := h.service.GetMyByID(c.Request.Context(), user.UserID, user.PrimaryRole(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	body := gin.H{"order": toOrderResponse(order, true)}
	if payment != nil {
		body["latest_payment"] = gin.H{
			"id":           payment.ID,
			"provider":     payment.Provider,
			"provider_ref": payment.ProviderTransactionID,
			"amount":       payment.Amount,
			"currency":     payment.Currency,
			"status":       payment.Status,
			"initiated_at": payment.InitiatedAt,
		}
	}
	response.JSON(c, http.StatusOK, body)
}

func (h *Handler) UpdateMyByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	order, err := h.service.UpdateDraft(c.Request.Context(), user.UserID, user.PrimaryRole(), c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toOrderResponse(order, true))
}

func (h *Handler) DeleteMyByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	if err := h.service.DeleteMy(c.Request.Context(), user.UserID, user.PrimaryRole(), c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) Submit(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	order, paymentTx, payment, err := h.service.Submit(c.Request.Context(), user.UserID, user.PrimaryRole(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, SubmitOrderResponse{
		Order: toOrderResponse(order, true),
		Payment: SubmitPaymentNextStep{
			TransactionID: paymentTx.ID,
			Provider:      paymentTx.Provider,
			Status:        payment.Status,
			Amount:        paymentTx.Amount,
			Currency:      paymentTx.Currency,
			CheckoutURL:   payment.RedirectURL,
			ProviderRef:   payment.TransactionID,
		},
	})
}

func (h *Handler) Cancel(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	order, err := h.service.Cancel(c.Request.Context(), user.UserID, user.PrimaryRole(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toOrderResponse(order, true))
}

func (h *Handler) ListPublic(c *gin.Context) {
	var budgetMin, budgetMax *float64
	if v := strings.TrimSpace(c.Query("budget_min")); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid budget_min")
			return
		}
		budgetMin = &parsed
	}
	if v := strings.TrimSpace(c.Query("budget_max")); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid budget_max")
			return
		}
		budgetMax = &parsed
	}
	var deadlineBefore *time.Time
	if v := strings.TrimSpace(c.Query("deadline_before")); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.JSONError(c, http.StatusBadRequest, "bad_request", "deadline_before must be RFC3339")
			return
		}
		deadlineBefore = &parsed
	}
	query := PublicOrdersQuery{
		CategorySlug:   strings.TrimSpace(c.Query("category")),
		BudgetMin:      budgetMin,
		BudgetMax:      budgetMax,
		DeadlineBefore: deadlineBefore,
		Q:              strings.TrimSpace(c.Query("q")),
		Region:         strings.TrimSpace(c.Query("region")),
		Page:           parseIntDefault(c.Query("page"), 1),
		PageSize:       parseIntDefault(c.Query("page_size"), 20),
	}
	if query.Page < 1 || query.PageSize < 1 || query.PageSize > 100 {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid pagination")
		return
	}
	items, total, err := h.service.ListPublic(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, OrdersListResponse{Items: toOrderResponses(items, false), Page: query.Page, PageSize: query.PageSize, Total: total})
}

func (h *Handler) GetByID(c *gin.Context) {
	var userID string
	var role string
	if user, ok := middleware.CurrentUser(c); ok {
		userID = user.UserID
		role = user.PrimaryRole()
	}
	order, err := h.service.GetByID(c.Request.Context(), c.Param("id"), userID, role)
	if err != nil {
		h.handleError(c, err)
		return
	}
	includeClient := role == "admin" || userID == order.ClientID
	response.JSON(c, http.StatusOK, toOrderResponse(order, includeClient))
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidRole):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Insufficient role")
	case errors.Is(err, ErrForbidden):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, ErrOrderNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Order not found")
	case errors.Is(err, ErrInvalidStatusTransition):
		response.JSONError(c, http.StatusConflict, "invalid_status_transition", "Invalid status transition")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid order data")
	case errors.Is(err, ErrInsufficientBalance):
		response.JSONError(c, http.StatusConflict, "insufficient_balance", "Insufficient wallet balance")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func toOrderResponse(o Order, includeClient bool) OrderResponse {
	res := OrderResponse{
		ID:               o.ID,
		CategoryID:       o.CategoryID,
		CategorySlug:     o.CategorySlug,
		CategoryName:     o.CategoryName,
		Title:            o.Title,
		Description:      o.Description,
		BudgetAmount:     o.BudgetAmount,
		Currency:         o.Currency,
		DeadlineAt:       o.DeadlineAt,
		Region:           o.Region,
		Promotions:       o.PromotionOptions,
		PostingFee:       o.PostingFee,
		PromotionFee:     o.PromotionFee,
		EscrowAmount:     o.EscrowAmount,
		TotalCharge:      o.TotalCharge,
		PaymentStatus:    o.PaymentStatus,
		PromotedUntil:    o.PromotedUntil,
		PinnedUntil:      o.PinnedUntil,
		HighlightedUntil: o.HighlightedUntil,
		Status:           o.Status,
		PublishedAt:      o.PublishedAt,
		CancelledAt:      o.CancelledAt,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}
	if includeClient {
		res.ClientID = o.ClientID
	}
	return res
}

func toOrderResponses(items []Order, includeClient bool) []OrderResponse {
	out := make([]OrderResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toOrderResponse(item, includeClient))
	}
	return out
}

func parseIntDefault(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	p, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return p
}
