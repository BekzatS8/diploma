package notifications

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

func (h *Handler) ListMy(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListQuery{
		Page:       parseIntDefault(c.Query("page"), 1),
		PageSize:   parseIntDefault(c.Query("page_size"), 20),
		Status:     strings.TrimSpace(c.Query("status")),
		Type:       strings.TrimSpace(c.Query("type")),
		UnreadOnly: parseBool(c.Query("unread_only")),
	}
	items, total, err := h.service.ListMy(c.Request.Context(), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": q.Page, "page_size": q.PageSize, "total": total})
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
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) MarkRead(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.MarkRead(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	updated, err := h.service.MarkAllRead(c.Request.Context(), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"updated": updated})
}

func (h *Handler) ListAdmin(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListQuery{
		Page:     parseIntDefault(c.Query("page"), 1),
		PageSize: parseIntDefault(c.Query("page_size"), 20),
		UserID:   strings.TrimSpace(c.Query("user_id")),
		Type:     strings.TrimSpace(c.Query("type")),
		Status:   strings.TrimSpace(c.Query("status")),
		Channel:  strings.TrimSpace(c.Query("channel")),
	}
	items, total, err := h.service.ListAdmin(c.Request.Context(), user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": q.Page, "page_size": q.PageSize, "total": total})
}

func (h *Handler) GetAdminByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetAdminByID(c.Request.Context(), user.PrimaryRole(), c.Param("id"))
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Notification not found")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func parseIntDefault(v string, def int) int {
	if strings.TrimSpace(v) == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func parseBool(v string) bool {
	if strings.TrimSpace(v) == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}
