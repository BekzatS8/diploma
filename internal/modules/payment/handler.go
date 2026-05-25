package payment

import (
	"errors"
	"io"
	"net/http"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"
	ordersmodule "buhpro/internal/modules/orders"

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
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	out, err := h.service.Create(c.Request.Context(), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleCreateErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, out)
}

func (h *Handler) Webhook(c *gin.Context) {
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Cannot read webhook payload")
		return
	}
	if err := h.service.HandleWebhook(c.Request.Context(), payload); err != nil {
		response.JSONError(c, http.StatusInternalServerError, "webhook_processing_failed", "Webhook processing failed")
		return
	}
	response.JSON(c, http.StatusOK, response.StatusResponse{Status: "ok"})
}

func (h *Handler) handleCreateErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidRole), errors.Is(err, ordersmodule.ErrInvalidRole):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Insufficient role")
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ordersmodule.ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid payment data")
	case errors.Is(err, ordersmodule.ErrOrderNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Order not found")
	case errors.Is(err, ordersmodule.ErrInvalidStatusTransition):
		response.JSONError(c, http.StatusConflict, "invalid_status_transition", "Invalid order status")
	default:
		response.JSONError(c, http.StatusInternalServerError, "payment_create_failed", "Payment creation failed")
	}
}
