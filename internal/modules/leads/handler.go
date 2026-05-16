package leads

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

const maxLeadMultipartMemory = 32 * 1024 * 1024

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SubmitExecutor(c *gin.Context) {
	req, err := h.parseSubmitRequest(c)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid executor lead payload")
		return
	}
	req.IPAddress = clientIP(c)
	req.UserAgent = c.Request.UserAgent()
	if req.UTMSource == "" {
		req.UTMSource = c.Query("utm_source")
	}
	if req.UTMMedium == "" {
		req.UTMMedium = c.Query("utm_medium")
	}
	if req.UTMCampaign == "" {
		req.UTMCampaign = c.Query("utm_campaign")
	}

	lead, err := h.service.SubmitExecutor(c.Request.Context(), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, ExecutorLeadSubmittedResponse{
		LeadID:  lead.ID,
		Status:  string(lead.Status),
		Message: "Заявка исполнителя отправлена на проверку",
	})
}

func (h *Handler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	items, total, err := h.service.List(c.Request.Context(), c.Query("status"), page, pageSize)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, ExecutorLeadListResponse{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (h *Handler) GetByID(c *gin.Context) {
	item, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	if err := h.service.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status, user.UserID, req.Notes); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) Approve(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req ApproveRequest
	_ = c.ShouldBindJSON(&req)
	userID, err := h.service.Approve(c.Request.Context(), c.Param("id"), user.UserID, req.Notes)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, ApproveResponse{Status: "converted", UserID: userID})
}

func (h *Handler) Reject(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}
	if err := h.service.Reject(c.Request.Context(), c.Param("id"), user.UserID, req.Reason); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "rejected"})
}

func (h *Handler) parseSubmitRequest(c *gin.Context) (SubmitExecutorLeadRequest, error) {
	if strings.HasPrefix(c.ContentType(), "multipart/") || c.ContentType() == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(maxLeadMultipartMemory); err != nil {
			return SubmitExecutorLeadRequest{}, err
		}
		req := SubmitExecutorLeadRequest{
			Email:           c.PostForm("email"),
			Password:        c.PostForm("password"),
			FirstName:       c.PostForm("first_name"),
			LastName:        c.PostForm("last_name"),
			MiddleName:      c.PostForm("middle_name"),
			IIN:             c.PostForm("iin"),
			Phone:           c.PostForm("phone"),
			City:            c.PostForm("city"),
			ExperienceLevel: c.PostForm("experience_level"),
			Specializations: parseSpecializations(c),
			Education:       c.PostForm("education"),
			WorkFormat:      c.PostForm("work_format"),
			HourlyRate:      parseOptionalFloat(c.PostForm("hourly_rate")),
			About:           firstNonEmpty(c.PostForm("about"), c.PostForm("bio")),
			TermsAccepted:   parseBool(c.PostForm("terms_accepted")),
			Source:          c.PostForm("source"),
			UTMSource:       c.PostForm("utm_source"),
			UTMMedium:       c.PostForm("utm_medium"),
			UTMCampaign:     c.PostForm("utm_campaign"),
		}
		if c.Request.MultipartForm != nil {
			req.Documents = collectDocuments(c.Request.MultipartForm.File)
		}
		return req, nil
	}

	var body struct {
		Email           string   `json:"email"`
		Password        string   `json:"password"`
		FirstName       string   `json:"first_name"`
		LastName        string   `json:"last_name"`
		MiddleName      string   `json:"middle_name"`
		IIN             string   `json:"iin"`
		Phone           string   `json:"phone"`
		City            string   `json:"city"`
		ExperienceLevel string   `json:"experience_level"`
		Specializations []string `json:"specializations"`
		Education       string   `json:"education"`
		WorkFormat      string   `json:"work_format"`
		HourlyRate      *float64 `json:"hourly_rate"`
		About           string   `json:"about"`
		TermsAccepted   bool     `json:"terms_accepted"`
		Source          string   `json:"source"`
		UTMSource       string   `json:"utm_source"`
		UTMMedium       string   `json:"utm_medium"`
		UTMCampaign     string   `json:"utm_campaign"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return SubmitExecutorLeadRequest{}, err
	}
	return SubmitExecutorLeadRequest{
		Email:           body.Email,
		Password:        body.Password,
		FirstName:       body.FirstName,
		LastName:        body.LastName,
		MiddleName:      body.MiddleName,
		IIN:             body.IIN,
		Phone:           body.Phone,
		City:            body.City,
		ExperienceLevel: body.ExperienceLevel,
		Specializations: body.Specializations,
		Education:       body.Education,
		WorkFormat:      body.WorkFormat,
		HourlyRate:      body.HourlyRate,
		About:           body.About,
		TermsAccepted:   body.TermsAccepted,
		Source:          body.Source,
		UTMSource:       body.UTMSource,
		UTMMedium:       body.UTMMedium,
		UTMCampaign:     body.UTMCampaign,
	}, nil
}

func collectDocuments(files map[string][]*multipart.FileHeader) []IncomingDocument {
	docs := make([]IncomingDocument, 0)
	add := func(field string, documentType DocumentType) {
		for _, header := range files[field] {
			file, err := header.Open()
			if err != nil {
				continue
			}
			docs = append(docs, IncomingDocument{Type: documentType, Filename: header.Filename, Reader: file})
		}
	}
	add("identity_document", DocumentIdentity)
	add("identity_documents", DocumentIdentity)
	add("education_document", DocumentEducation)
	add("education_documents", DocumentEducation)
	add("ip_registration_document", DocumentIPRegistration)
	add("ip_registration_documents", DocumentIPRegistration)
	add("documents", DocumentOther)
	return docs
}

func parseSpecializations(c *gin.Context) []string {
	items := c.PostFormArray("specializations")
	raw := strings.TrimSpace(c.PostForm("specializations"))
	if raw != "" {
		if strings.HasPrefix(raw, "[") {
			var parsed []string
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				items = append(items, parsed...)
			}
		} else {
			for _, part := range strings.Split(raw, ",") {
				items = append(items, part)
			}
		}
	}
	return items
}

func parseOptionalFloat(value string) *float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parsePagination(c *gin.Context) (int, int) {
	page := 1
	if value := strings.TrimSpace(c.Query("page")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 20
	if value := strings.TrimSpace(c.Query("page_size")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	return page, pageSize
}

func clientIP(c *gin.Context) string {
	if value := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); value != "" {
		parts := strings.Split(value, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return c.ClientIP()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "not_found", "Lead not found")
	case errors.Is(err, ErrEmailExists):
		response.JSONError(c, http.StatusConflict, "email_exists", "Email already in use")
	case errors.Is(err, ErrDuplicate):
		response.JSONError(c, http.StatusConflict, "lead_exists", "Open lead already exists for this email")
	case errors.Is(err, ErrAlreadyConverted):
		response.JSONError(c, http.StatusConflict, "already_converted", "Lead already converted")
	case errors.Is(err, ErrInvalidStatus):
		response.JSONError(c, http.StatusConflict, "invalid_status", "Invalid lead status")
	case errors.Is(err, ErrDocumentRequired):
		response.JSONError(c, http.StatusBadRequest, "documents_required", "Identity and education documents are required")
	case errors.Is(err, ErrDocumentTooLarge):
		response.JSONError(c, http.StatusRequestEntityTooLarge, "document_too_large", "Document exceeds 5 MB")
	case errors.Is(err, ErrInvalidMime):
		response.JSONError(c, http.StatusUnprocessableEntity, "invalid_file_type", "Only PDF, JPG and PNG documents are allowed")
	case errors.Is(err, ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid lead data")
	default:
		if err != nil && err.Error() == "password must be at least 8 characters" {
			response.JSONError(c, http.StatusBadRequest, "invalid_password", err.Error())
			return
		}
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
