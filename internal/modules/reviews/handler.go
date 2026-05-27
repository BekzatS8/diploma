package reviews

import (
	"errors"
	"net/http"
	"strconv"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type createReq struct {
	Rating  int     `json:"rating" binding:"required,min=1,max=5"`
	Comment *string `json:"comment"`
}

type createEntityReq struct {
	TargetType string         `json:"target_type" binding:"required"`
	TargetID   string         `json:"target_id" binding:"required,uuid"`
	Rating     int            `json:"rating" binding:"required,min=1,max=5"`
	Comment    *string        `json:"comment"`
	Metadata   map[string]any `json:"metadata"`
}

func (h *Handler) Create(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.Create(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), req.Rating, req.Comment)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) CreateEntity(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req createEntityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.CreateEntity(c.Request.Context(), user.UserID, user.PrimaryRole(), req.TargetType, req.TargetID, req.Rating, req.Comment, req.Metadata)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) ListByTarget(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.service.ListByTarget(c.Request.Context(), c.Query("target_type"), c.Query("target_id"), page, size)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": page, "page_size": size, "total": total})
}

func (h *Handler) GetRatingSummary(c *gin.Context) {
	item, err := h.service.GetRatingSummary(c.Request.Context(), c.Query("target_type"), c.Query("target_id"))
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetByOrder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetByOrder(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ListExecutor(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.service.ListExecutor(c.Request.Context(), c.Param("executorId"), page, size)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": page, "page_size": size, "total": total})
}

func (h *Handler) ListUser(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.service.ListUser(c.Request.Context(), c.Param("userId"), page, size)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": page, "page_size": size, "total": total})
}

func (h *Handler) ListAuthored(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.service.ListAuthored(c.Request.Context(), user.UserID, page, size)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"items": items, "page": page, "page_size": size, "total": total})
}

func (h *Handler) GetAuthored(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetAuthored(c.Request.Context(), c.Param("reviewId"), user.UserID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) PatchAuthored(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.UpdateAuthored(c.Request.Context(), c.Param("reviewId"), user.UserID, req.Rating, req.Comment)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteAuthored(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.DeleteAuthored(c.Request.Context(), c.Param("reviewId"), user.UserID)
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
		response.JSONError(c, http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, ErrAlreadyExists):
		response.JSONError(c, http.StatusConflict, "already_exists", "Review already exists")
	case errors.Is(err, ErrInvalidState):
		response.JSONError(c, http.StatusConflict, "invalid_state", "Review preconditions are not met")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid review data")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
