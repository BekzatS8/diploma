package adminusers

import (
	"errors"
	"net/http"
	"strconv"

	"buhpro/internal/common/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListExecutors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.service.ListExecutors(c.Request.Context(), c.Query("q"), page, pageSize)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to list executors")
		return
	}
	response.JSON(c, http.StatusOK, resp)
}

func (h *Handler) PromoteToCoach(c *gin.Context) {
	userID := c.Param("userId")
	resp, err := h.service.PromoteExecutorToCoach(c.Request.Context(), userID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, resp)
}

func (h *Handler) RevokeCoach(c *gin.Context) {
	userID := c.Param("userId")
	resp, err := h.service.RevokeCoachFromExecutor(c.Request.Context(), userID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, resp)
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "User not found")
	case errors.Is(err, ErrNotExecutor):
		response.JSONError(c, http.StatusBadRequest, "not_executor", "Only active executors can be promoted to coach")
	case errors.Is(err, ErrAlreadyCoach):
		response.JSONError(c, http.StatusConflict, "already_coach", "User already has coach capabilities")
	case errors.Is(err, ErrNotCoach):
		response.JSONError(c, http.StatusBadRequest, "not_coach", "User does not have coach capabilities")
	default:
		if err != nil && err.Error() == "cannot promote inactive user" {
			response.JSONError(c, http.StatusBadRequest, "inactive_user", "Cannot promote inactive user")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to promote user")
	}
}
