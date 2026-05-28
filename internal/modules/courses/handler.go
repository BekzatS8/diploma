package courses

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

func (h *Handler) CreateCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.CreateCourse(c.Request.Context(), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) PatchCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.UpdateCourse(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetCoachCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, materials, err := h.service.GetCoachCourse(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, CourseDetailResponse{Course: item, Materials: materials})
}

func (h *Handler) ListCoachCourses(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListCoursesQuery{
		Status:   strings.TrimSpace(c.Query("status")),
		Category: strings.TrimSpace(c.Query("category")),
		Search:   strings.TrimSpace(c.Query("q")),
		Page:     parseIntDefault(c.Query("page"), 1),
		PageSize: parseIntDefault(c.Query("page_size"), 20),
	}
	items, total, err := h.service.ListCoachCourses(c.Request.Context(), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.ListEnvelope[Course]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total})
}

func (h *Handler) CreatorAnalytics(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.CreatorAnalytics(c.Request.Context(), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ListCourseStudents(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	page := parseIntDefault(c.Query("page"), 1)
	size := parseIntDefault(c.Query("page_size"), 20)
	items, total, err := h.service.ListCourseStudents(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), page, size)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.ListEnvelope[CourseStudent]{Items: items, Page: page, PageSize: size, Total: total})
}

func (h *Handler) PublishCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.PublishCourse(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ArchiveCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.ArchiveCourse(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	if err := h.service.DeleteCourse(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole()); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.StatusResponse{Status: "deleted"})
}

func (h *Handler) CreateMaterial(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req CreateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.AddMaterial(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) PatchMaterial(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req UpdateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.UpdateMaterial(c.Request.Context(), c.Param("id"), c.Param("materialId"), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteMaterial(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	if err := h.service.DeleteMaterial(c.Request.Context(), c.Param("id"), c.Param("materialId"), user.UserID, user.PrimaryRole()); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.StatusResponse{Status: "deleted"})
}

func (h *Handler) ListCourses(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListCoursesQuery{
		Status:           strings.TrimSpace(c.Query("status")),
		ModerationStatus: strings.TrimSpace(c.Query("moderation_status")),
		Category:         strings.TrimSpace(c.Query("category")),
		Search:           strings.TrimSpace(c.Query("q")),
		Page:             parseIntDefault(c.Query("page"), 1),
		PageSize:         parseIntDefault(c.Query("page_size"), 20),
	}
	items, total, err := h.service.ListCoursesForRole(c.Request.Context(), user.UserID, user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.ListEnvelope[Course]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total})
}

func (h *Handler) ListAdminCourses(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListCoursesQuery{
		Status:           strings.TrimSpace(c.Query("status")),
		ModerationStatus: strings.TrimSpace(c.Query("moderation_status")),
		Category:         strings.TrimSpace(c.Query("category")),
		Search:           strings.TrimSpace(c.Query("q")),
		Page:             parseIntDefault(c.Query("page"), 1),
		PageSize:         parseIntDefault(c.Query("page_size"), 20),
	}
	items, total, err := h.service.ListAdminCourses(c.Request.Context(), user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.ListEnvelope[Course]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total})
}

func (h *Handler) ApproveCourseModeration(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.ApproveCourseModeration(c.Request.Context(), c.Param("id"), user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) RejectCourseModeration(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req RejectCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.RejectCourseModeration(c.Request.Context(), c.Param("id"), user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, materials, err := h.service.GetCourseCatalogByID(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, CourseDetailResponse{Course: item, Materials: materials})
}

func (h *Handler) CreateAssignment(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, err := h.service.CreateAssignment(c.Request.Context(), user.UserID, user.PrimaryRole(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) ListAdminAssignments(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	q := ListAssignmentsQuery{
		ExecutorID: strings.TrimSpace(c.Query("executor_id")),
		CourseID:   strings.TrimSpace(c.Query("course_id")),
		Status:     strings.TrimSpace(c.Query("status")),
		Source:     strings.TrimSpace(c.Query("source")),
		Page:       parseIntDefault(c.Query("page"), 1),
		PageSize:   parseIntDefault(c.Query("page_size"), 20),
	}
	items, total, err := h.service.ListAdminAssignments(c.Request.Context(), user.PrimaryRole(), q)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.ListEnvelope[CourseAssignment]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total})
}

func (h *Handler) EnrollMyCourse(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req EnrollCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	item, created, err := h.service.EnrollMyCourse(c.Request.Context(), user.UserID, user.PrimaryRole(), req.CourseID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if created {
		response.JSON(c, http.StatusCreated, item)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ListMyAssignments(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	page := parseIntDefault(c.Query("page"), 1)
	size := parseIntDefault(c.Query("page_size"), 20)
	items, total, err := h.service.ListMyAssignments(c.Request.Context(), user.UserID, user.PrimaryRole(), page, size)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.ListEnvelope[CourseAssignment]{Items: items, Page: page, PageSize: size, Total: total})
}

func (h *Handler) GetMyAssignment(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.GetMyAssignmentByID(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) MarkCompleted(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.MarkCompleted(c.Request.Context(), c.Param("id"), user.UserID, user.PrimaryRole())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) MarkMaterialCompleted(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	item, err := h.service.MarkMaterialCompleted(c.Request.Context(), c.Param("id"), c.Param("materialId"), user.UserID, user.PrimaryRole())
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
	case errors.Is(err, ErrConflict):
		response.JSONError(c, http.StatusConflict, "conflict", "Conflict")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid input")
	default:
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func parseIntDefault(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	p, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return p
}
