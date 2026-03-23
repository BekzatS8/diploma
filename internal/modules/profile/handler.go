package profile

import (
	"net/http"

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

func (h *Handler) Get(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	profile, err := h.service.GetCurrentProfile(c.Request.Context(), user.UserID, user.PrimaryRole())
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load profile")
		return
	}

	response.JSON(c, http.StatusOK, profile)
}

func (h *Handler) Patch(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	if err := h.service.UpdateCurrentProfile(c.Request.Context(), user.UserID, user.PrimaryRole(), req); err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to update profile")
		return
	}

	profile, err := h.service.GetCurrentProfile(c.Request.Context(), user.UserID, user.PrimaryRole())
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load profile")
		return
	}

	response.JSON(c, http.StatusOK, profile)
}
