package responses

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req CreateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.Create(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toView(item, true))
}

func (h *Handler) ListOrderMy(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q, message, ok := parseListQuery(c)
	if !ok {
		response.JSONError(c, http.StatusBadRequest, "bad_request", message)
		return
	}
	items, total, err := h.service.ListOrderMy(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, ListResponse{Items: toViews(items, true), Page: q.Page, PageSize: q.PageSize, Total: total})
}

func (h *Handler) GetOrderMyByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetOrderMyByID(c.Request.Context(), c.Param("id"), c.Param("responseId"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toView(item, true))
}

func (h *Handler) UpdateOrderMyByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req UpdateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.UpdateDraft(c.Request.Context(), c.Param("id"), c.Param("responseId"), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toView(item, true))
}

func (h *Handler) DeleteOrderMyByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.Param("id"), c.Param("responseId"), user.UserID, user.PrimaryRole()); err != nil {
		h.handleErr(c, err)
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
	item, payTx, pay, err := h.service.Submit(c.Request.Context(), c.Param("id"), c.Param("responseId"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, SubmitResponsePayload{Response: toView(item, true), Payment: SubmitPaymentNextStep{TransactionID: payTx.ID, Provider: payTx.Provider, Status: pay.Status, Amount: payTx.Amount, Currency: payTx.Currency, CheckoutURL: pay.RedirectURL, ProviderRef: pay.TransactionID}})
}

func (h *Handler) Cancel(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.Cancel(c.Request.Context(), c.Param("id"), c.Param("responseId"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toView(item, true))
}

func (h *Handler) ListMy(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q, message, ok := parseListQuery(c)
	if !ok {
		response.JSONError(c, http.StatusBadRequest, "bad_request", message)
		return
	}
	items, total, err := h.service.ListMy(c.Request.Context(), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, ListResponse{Items: toViews(items, true), Page: q.Page, PageSize: q.PageSize, Total: total})
}

func (h *Handler) GetMyByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetMyByID(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	includeExecutor := user.PrimaryRole() == "executor" || user.PrimaryRole() == "admin"
	response.JSON(c, http.StatusOK, toView(item, includeExecutor))
}

func (h *Handler) ListClientOrder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q, message, ok := parseListQuery(c)
	if !ok {
		response.JSONError(c, http.StatusBadRequest, "bad_request", message)
		return
	}
	items, total, err := h.service.ListClientOrder(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, ListResponse{Items: toViews(items, true), Page: q.Page, PageSize: q.PageSize, Total: total})
}

func (h *Handler) GetClientOrderByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetClientOrderByID(c.Request.Context(), c.Param("id"), c.Param("responseId"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, toView(item, true))
}

func parseListQuery(c *gin.Context) (ListQuery, string, bool) {
	page := 1
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			return ListQuery{}, "Invalid page", false
		}
		page = p
	}
	size := 20
	if v := strings.TrimSpace(c.Query("page_size")); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 100 {
			return ListQuery{}, "Invalid page_size", false
		}
		size = p
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && !IsKnownStatus(status) {
		return ListQuery{}, "Invalid status", false
	}
	return ListQuery{Status: status, Page: page, PageSize: size}, "", true
}

func toView(r Response, includeExecutor bool) ResponseView {
	v := ResponseView{ID: r.ID, OrderID: r.OrderID, CoverLetter: r.CoverLetter, ProposedAmount: r.ProposedAmount, Currency: r.Currency, Status: r.Status, IsPaid: r.IsPaid, PaidAt: r.PaidAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, OrderTitle: r.OrderTitle}
	if includeExecutor {
		v.ExecutorID = r.ExecutorID
	}
	return v
}
func toViews(items []Response, includeExecutor bool) []ResponseView {
	out := make([]ResponseView, 0, len(items))
	for _, it := range items {
		out = append(out, toView(it, includeExecutor))
	}
	return out
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidRole):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Insufficient role")
	case errors.Is(err, ErrForbidden):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Response or order not found")
	case errors.Is(err, ErrInvalidStatus):
		response.JSONError(c, http.StatusConflict, "invalid_status_transition", "Invalid status transition")
	case errors.Is(err, ErrDuplicate):
		response.JSONError(c, http.StatusConflict, "already_exists", "Executor already has response for this order")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid response payload")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
