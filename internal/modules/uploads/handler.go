package uploads

import (
	"errors"
	"net/http"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

const maxMultipartMemory = 32 * 1024 * 1024

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Upload(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	if err := c.Request.ParseMultipartForm(maxMultipartMemory); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Failed to parse multipart form")
		return
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) == 0 {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Missing file field")
		return
	}

	items := make([]UploadView, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			response.JSONError(c, http.StatusBadRequest, "bad_request", "Failed to open uploaded file")
			return
		}
		item, err := h.service.Upload(c.Request.Context(), user.UserID, header.Filename, file)
		_ = file.Close()
		if err != nil {
			h.handleErr(c, err)
			return
		}
		items = append(items, h.service.ToView(item, true))
	}

	response.JSON(c, http.StatusCreated, UploadListResponse{Items: items})
}

func (h *Handler) GetByID(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetByIDForUser(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, h.service.ToView(item, true))
}

func (h *Handler) ListMy(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	items, err := h.service.ListByAuthor(c.Request.Context(), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	views := make([]UploadView, 0, len(items))
	for _, item := range items {
		views = append(views, h.service.ToView(item, true))
	}
	response.JSON(c, http.StatusOK, UploadListResponse{Items: views})
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

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Upload not found")
	case errors.Is(err, ErrForbidden):
		response.JSONError(c, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, ErrFileTooLarge):
		response.JSONError(c, http.StatusRequestEntityTooLarge, "file_too_large", "File exceeds 50 MB")
	case errors.Is(err, ErrInvalidMime):
		response.JSONError(c, http.StatusUnprocessableEntity, "invalid_file_type", "File type is not allowed")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid upload")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
