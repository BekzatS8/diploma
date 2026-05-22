package attachments

import (
	"errors"
	"net/http"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"
	uploadsmodule "buhpro/internal/modules/uploads"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Attach(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req AttachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	items, err := h.service.Attach(c.Request.Context(), req.UploadIDs, user.UserID, user.PrimaryRole(), TargetType(req.TargetType), req.TargetID, req.Metadata)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, AttachmentListResponse{Items: items})
}

func (h *Handler) List(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	items, err := h.service.ListByTarget(c.Request.Context(), TargetType(c.Query("target_type")), c.Query("target_id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, AttachmentListResponse{Items: items})
}

func (h *Handler) Delete(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole()); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) Reorder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	if err := h.service.Reorder(c.Request.Context(), req.IDs, user.UserID, user.PrimaryRole()); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid attachment data")
	case errors.Is(err, ErrForbidden):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, ErrNotFound), errors.Is(err, uploadsmodule.ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Attachment or upload not found")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
