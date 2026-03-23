package devpayments

import (
	"errors"
	"net/http"

	"buhpro/internal/common/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Confirm(c *gin.Context) {
	if err := h.service.Confirm(c.Request.Context(), c.Param("transactionId")); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "confirmed"})
}

func (h *Handler) Fail(c *gin.Context) {
	if err := h.service.Fail(c.Request.Context(), c.Param("transactionId")); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "failed"})
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrDevDisabled):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Dev payment endpoints are disabled")
	default:
		response.JSONError(c, http.StatusConflict, "payment_state_error", err.Error())
	}
}
