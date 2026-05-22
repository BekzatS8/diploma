package orders

import (
	"errors"
	"net/http"
	"net/url"
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
	page, message, ok := parsePositiveInt(c.Request.URL.Query(), "page", 1, 0)
	if !ok {
		response.JSONError(c, http.StatusBadRequest, "bad_request", message)
		return
	}
	pageSize, message, ok := parsePositiveInt(c.Request.URL.Query(), "page_size", 20, 100)
	if !ok {
		response.JSONError(c, http.StatusBadRequest, "bad_request", message)
		return
	}
	query := MyOrdersQuery{
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	}
	if query.Status != "" && !IsKnownStatus(query.Status) {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid status")
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
	body := OrderDetailsResponse{Order: toOrderResponse(order, true)}
	if payment != nil {
		body.LatestPayment = &PaymentTransactionResponse{
			ID:          payment.ID,
			Provider:    payment.Provider,
			ProviderRef: payment.ProviderTransactionID,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
			Status:      payment.Status,
			InitiatedAt: payment.InitiatedAt,
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
	response.JSON(c, http.StatusOK, response.StatusResponse{Status: "deleted"})
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
	query, message, ok := parsePublicOrdersQuery(c.Request.URL.Query())
	if !ok {
		response.JSONError(c, http.StatusBadRequest, "bad_request", message)
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

func parsePublicOrdersQuery(values url.Values) (PublicOrdersQuery, string, bool) {
	budgetMin, message, ok := parseOptionalNonNegativeFloat(values, "budget_min")
	if !ok {
		return PublicOrdersQuery{}, message, false
	}
	budgetMax, message, ok := parseOptionalNonNegativeFloat(values, "budget_max")
	if !ok {
		return PublicOrdersQuery{}, message, false
	}
	if budgetMin != nil && budgetMax != nil && *budgetMin > *budgetMax {
		return PublicOrdersQuery{}, "budget_min must be less than or equal to budget_max", false
	}

	var deadlineBefore *time.Time
	if v := strings.TrimSpace(values.Get("deadline_before")); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return PublicOrdersQuery{}, "deadline_before must be RFC3339", false
		}
		deadlineBefore = &parsed
	}

	page, message, ok := parsePositiveInt(values, "page", 1, 0)
	if !ok {
		return PublicOrdersQuery{}, message, false
	}
	pageSize, message, ok := parsePositiveInt(values, "page_size", 20, 100)
	if !ok {
		return PublicOrdersQuery{}, message, false
	}

	category := strings.ToLower(strings.TrimSpace(values.Get("category")))
	region := strings.TrimSpace(values.Get("region"))
	q := strings.TrimSpace(values.Get("q"))
	if len(category) > 100 {
		return PublicOrdersQuery{}, "category is too long", false
	}
	if len(region) > 100 {
		return PublicOrdersQuery{}, "region is too long", false
	}
	if len(q) > 200 {
		return PublicOrdersQuery{}, "q is too long", false
	}

	return PublicOrdersQuery{
		CategorySlug:   category,
		BudgetMin:      budgetMin,
		BudgetMax:      budgetMax,
		DeadlineBefore: deadlineBefore,
		Region:         region,
		Q:              q,
		Page:           page,
		PageSize:       pageSize,
	}, "", true
}

func parseOptionalNonNegativeFloat(values url.Values, name string) (*float64, string, bool) {
	raw := strings.TrimSpace(values.Get(name))
	if raw == "" {
		return nil, "", true
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || parsed < 0 {
		return nil, "Invalid " + name, false
	}
	return &parsed, "", true
}

func parsePositiveInt(values url.Values, name string, def, max int) (int, string, bool) {
	raw := strings.TrimSpace(values.Get(name))
	if raw == "" {
		return def, "", true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 || (max > 0 && parsed > max) {
		return 0, "Invalid " + name, false
	}
	return parsed, "", true
}
