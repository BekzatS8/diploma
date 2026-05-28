package orderreports

import (
	"errors"
	"net/http"
	"strconv"

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
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Reason must be 10–2000 characters")
		return
	}
	resp, err := h.service.Create(c.Request.Context(), c.Param("id"), user.UserID, req.Reason)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, resp)
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resp, err := h.service.List(c.Request.Context(), c.Query("status"), page, pageSize)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, resp)
}

func (h *Handler) Dismiss(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req ReviewRequest
	_ = c.ShouldBindJSON(&req)
	resp, err := h.service.Dismiss(c.Request.Context(), c.Param("id"), user.UserID, req.Notes)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, resp)
}

func (h *Handler) RemoveOrder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req ReviewRequest
	_ = c.ShouldBindJSON(&req)
	resp, err := h.service.RemoveOrder(c.Request.Context(), c.Param("id"), user.UserID, req.Notes)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, resp)
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Report not found")
	case errors.Is(err, ErrOrderNotFound):
		response.JSONError(c, http.StatusNotFound, "order_not_found", "Order not found")
	case errors.Is(err, ErrOrderNotReportable):
		response.JSONError(c, http.StatusBadRequest, "order_not_reportable", "Only published active orders can be reported")
	case errors.Is(err, ErrDuplicatePending):
		response.JSONError(c, http.StatusConflict, "duplicate_report", "You already have a pending report for this order")
	case errors.Is(err, ErrAlreadyReviewed):
		response.JSONError(c, http.StatusConflict, "already_reviewed", "Report has already been processed")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Request failed")
	}
}
