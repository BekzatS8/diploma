package chats

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

func (h *Handler) ListMyChats(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListChatsQuery{Page: parseIntDefault(c.Query("page"), 1), PageSize: parseIntDefault(c.Query("page_size"), 20)}
	items, total, err := h.service.ListMyChats(c.Request.Context(), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": q.Page, "page_size": q.PageSize, "total": total})
}

func (h *Handler) GetMyChatByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetMyChatByID(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ListMyMessages(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListMessagesQuery{Page: parseIntDefault(c.Query("page"), 1), PageSize: parseIntDefault(c.Query("page_size"), 20)}
	items, total, err := h.service.ListMessagesMy(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": q.Page, "page_size": q.PageSize, "total": total, "order": "asc"})
}

type sendMessageRequest struct {
	Text string `json:"text" binding:"required"`
}

func (h *Handler) SendMyMessage(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.SendMessageMy(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), req.Text)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) MarkMyChatRead(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	if err := h.service.MarkReadMy(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole()); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ListAdminChats(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	page := parseIntDefault(c.Query("page"), 1)
	size := parseIntDefault(c.Query("page_size"), 20)
	items, total, err := h.service.ListAdminChats(c.Request.Context(), user.PrimaryRole(), page, size)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": page, "page_size": size, "total": total})
}

func (h *Handler) GetAdminChatByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetAdminChatByID(c.Request.Context(), user.PrimaryRole(), c.Param("id"))
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ListAdminMessages(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListMessagesQuery{Page: parseIntDefault(c.Query("page"), 1), PageSize: parseIntDefault(c.Query("page_size"), 20)}
	items, total, err := h.service.ListAdminMessages(c.Request.Context(), user.PrimaryRole(), c.Param("id"), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": q.Page, "page_size": q.PageSize, "total": total, "order": "asc"})
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Chat or message not found")
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
