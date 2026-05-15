package profile

import (
	"errors"
	"net/http"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"
	"buhpro/internal/modules/uploads"

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

func (h *Handler) SetAvatar(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req SetAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	if err := h.service.SetAvatar(c.Request.Context(), user.UserID, user.PrimaryRole(), req.UploadID); err != nil {
		h.handleErr(c, err)
		return
	}

	profile, err := h.service.GetCurrentProfile(c.Request.Context(), user.UserID, user.PrimaryRole())
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load profile")
		return
	}
	response.JSON(c, http.StatusOK, profile)
}

func (h *Handler) ClearAvatar(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	if err := h.service.ClearAvatar(c.Request.Context(), user.UserID, user.PrimaryRole()); err != nil {
		h.handleErr(c, err)
		return
	}
	profile, err := h.service.GetCurrentProfile(c.Request.Context(), user.UserID, user.PrimaryRole())
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load profile")
		return
	}
	response.JSON(c, http.StatusOK, profile)
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, uploads.ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Upload not found")
	case errors.Is(err, uploads.ErrForbidden):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Forbidden")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
