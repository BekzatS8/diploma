package leads

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	authmodule "buhpro/internal/modules/auth"
	"buhpro/internal/platform/storage"

	"github.com/google/uuid"
)

const MaxLeadDocumentSize = 5 * 1024 * 1024

var allowedDocumentMIMETypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

type Service struct {
	repo    *Repository
	storage storage.Provider
}

func NewService(repo *Repository, storage storage.Provider) *Service {
	return &Service{repo: repo, storage: storage}
}

func (s *Service) SubmitExecutor(ctx context.Context, req SubmitExecutorLeadRequest) (ExecutorLead, error) {
	defer closeIncomingDocuments(req.Documents)

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.MiddleName = strings.TrimSpace(req.MiddleName)
	req.IIN = strings.TrimSpace(req.IIN)
	req.Phone = strings.TrimSpace(req.Phone)
	req.City = strings.TrimSpace(req.City)
	req.ExperienceLevel = strings.TrimSpace(req.ExperienceLevel)
	req.Education = strings.TrimSpace(req.Education)
	req.WorkFormat = strings.TrimSpace(req.WorkFormat)
	req.About = strings.TrimSpace(req.About)
	req.Specializations = normalizeSpecializations(req.Specializations)

	if req.Email == "" || req.FirstName == "" || req.LastName == "" || req.IIN == "" ||
		req.Phone == "" || req.City == "" || req.ExperienceLevel == "" || req.Education == "" ||
		req.About == "" || !req.TermsAccepted {
		return ExecutorLead{}, ErrInvalidInput
	}
	if len(req.IIN) != 12 {
		return ExecutorLead{}, ErrInvalidInput
	}
	if len(req.Specializations) == 0 {
		return ExecutorLead{}, ErrInvalidInput
	}
	if err := authmodule.ValidatePassword(req.Password); err != nil {
		return ExecutorLead{}, err
	}
	if exists, err := s.repo.UserEmailExists(ctx, req.Email); err != nil {
		return ExecutorLead{}, err
	} else if exists {
		return ExecutorLead{}, ErrEmailExists
	}
	if !hasIncomingDocument(req.Documents, DocumentIdentity) || !hasIncomingDocument(req.Documents, DocumentEducation) {
		return ExecutorLead{}, ErrDocumentRequired
	}

	hash, err := authmodule.HashPassword(req.Password)
	if err != nil {
		return ExecutorLead{}, err
	}

	now := time.Now()
	leadID := uuid.NewString()
	lead := ExecutorLead{
		ID:              leadID,
		Email:           req.Email,
		PasswordHash:    hash,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		MiddleName:      stringPtr(req.MiddleName),
		IIN:             req.IIN,
		Phone:           req.Phone,
		City:            req.City,
		ExperienceLevel: req.ExperienceLevel,
		Specializations: req.Specializations,
		Education:       req.Education,
		WorkFormat:      stringPtr(req.WorkFormat),
		HourlyRate:      req.HourlyRate,
		About:           req.About,
		TermsAccepted:   req.TermsAccepted,
		Status:          StatusNew,
		Source:          stringPtr(req.Source),
		UTMSource:       stringPtr(req.UTMSource),
		UTMMedium:       stringPtr(req.UTMMedium),
		UTMCampaign:     stringPtr(req.UTMCampaign),
		IPAddress:       stringPtr(req.IPAddress),
		UserAgent:       stringPtr(req.UserAgent),
		SubmittedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	docs := make([]ExecutorLeadDocument, 0, len(req.Documents))
	savedKeys := make([]string, 0, len(req.Documents))
	for _, incoming := range req.Documents {
		doc, err := s.saveDocument(ctx, leadID, incoming, now)
		if err != nil {
			for _, key := range savedKeys {
				_ = s.storage.Delete(ctx, key)
			}
			return ExecutorLead{}, err
		}
		savedKeys = append(savedKeys, doc.FilePath)
		docs = append(docs, doc)
	}

	if err := s.repo.Create(ctx, lead, docs); err != nil {
		for _, key := range savedKeys {
			_ = s.storage.Delete(ctx, key)
		}
		return ExecutorLead{}, err
	}
	lead.Documents = docs
	return lead, nil
}

func (s *Service) List(ctx context.Context, status string, page, pageSize int) ([]ExecutorLeadView, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var parsed Status
	if strings.TrimSpace(status) != "" {
		parsed = Status(strings.TrimSpace(status))
		if !isValidStatus(parsed) {
			return nil, 0, ErrInvalidStatus
		}
	}
	items, total, err := s.repo.List(ctx, parsed, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ExecutorLeadView, 0, len(items))
	for _, item := range items {
		views = append(views, s.ToView(item))
	}
	return views, total, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (ExecutorLeadView, error) {
	if _, err := uuid.Parse(id); err != nil {
		return ExecutorLeadView{}, ErrInvalidInput
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ExecutorLeadView{}, err
	}
	return s.ToView(item), nil
}

func (s *Service) UpdateStatus(ctx context.Context, id, status, adminID, notes string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidInput
	}
	if _, err := uuid.Parse(adminID); err != nil {
		return ErrInvalidInput
	}
	parsed := Status(strings.TrimSpace(status))
	if !isValidManualStatus(parsed) {
		return ErrInvalidStatus
	}
	return s.repo.UpdateStatus(ctx, id, parsed, adminID, strings.TrimSpace(notes))
}

func (s *Service) Reject(ctx context.Context, id, adminID, reason string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidInput
	}
	if _, err := uuid.Parse(adminID); err != nil {
		return ErrInvalidInput
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrInvalidInput
	}
	return s.repo.Reject(ctx, id, adminID, reason)
}

func (s *Service) Approve(ctx context.Context, id, adminID, notes string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", ErrInvalidInput
	}
	if _, err := uuid.Parse(adminID); err != nil {
		return "", ErrInvalidInput
	}
	return s.repo.ApproveAndConvert(ctx, id, adminID, strings.TrimSpace(notes))
}

func (s *Service) ToView(item ExecutorLead) ExecutorLeadView {
	docs := make([]ExecutorLeadDocumentView, 0, len(item.Documents))
	for _, doc := range item.Documents {
		docs = append(docs, ExecutorLeadDocumentView{
			ID:           doc.ID,
			DocumentType: string(doc.DocumentType),
			URL:          s.storage.URL(doc.FilePath),
			OriginalName: doc.OriginalName,
			MimeType:     doc.MimeType,
			SizeBytes:    doc.SizeBytes,
			CreatedAt:    doc.CreatedAt,
		})
	}
	return ExecutorLeadView{
		ID:              item.ID,
		Email:           item.Email,
		FirstName:       item.FirstName,
		LastName:        item.LastName,
		MiddleName:      item.MiddleName,
		IIN:             item.IIN,
		Phone:           item.Phone,
		City:            item.City,
		ExperienceLevel: item.ExperienceLevel,
		Specializations: item.Specializations,
		Education:       item.Education,
		WorkFormat:      item.WorkFormat,
		HourlyRate:      item.HourlyRate,
		About:           item.About,
		TermsAccepted:   item.TermsAccepted,
		Status:          string(item.Status),
		Priority:        item.Priority,
		Notes:           item.Notes,
		RejectionReason: item.RejectionReason,
		SubmittedAt:     item.SubmittedAt,
		ReviewedAt:      item.ReviewedAt,
		ReviewedBy:      item.ReviewedBy,
		ConvertedAt:     item.ConvertedAt,
		ConvertedUserID: item.ConvertedUserID,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		Documents:       docs,
	}
}

func (s *Service) saveDocument(ctx context.Context, leadID string, incoming IncomingDocument, now time.Time) (ExecutorLeadDocument, error) {
	if incoming.Reader == nil || !isValidDocumentType(incoming.Type) {
		return ExecutorLeadDocument{}, ErrInvalidInput
	}

	buf := &bytes.Buffer{}
	size, err := io.Copy(buf, io.LimitReader(incoming.Reader, MaxLeadDocumentSize+1))
	if err != nil {
		return ExecutorLeadDocument{}, fmt.Errorf("read document: %w", err)
	}
	if size == 0 {
		return ExecutorLeadDocument{}, ErrInvalidInput
	}
	if size > MaxLeadDocumentSize {
		return ExecutorLeadDocument{}, ErrDocumentTooLarge
	}

	mimeType := http.DetectContentType(buf.Bytes())
	mimeType, _, _ = mime.ParseMediaType(mimeType)
	if !allowedDocumentMIMETypes[mimeType] {
		return ExecutorLeadDocument{}, ErrInvalidMime
	}

	docID := uuid.NewString()
	ext := strings.ToLower(filepath.Ext(incoming.Filename))
	if ext == "" {
		if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
			ext = exts[0]
		}
	}
	key := fmt.Sprintf("leads/%s/%s%s", leadID, docID, ext)
	if _, err := s.storage.Upload(ctx, key, bytes.NewReader(buf.Bytes()), mimeType); err != nil {
		return ExecutorLeadDocument{}, fmt.Errorf("save document: %w", err)
	}

	return ExecutorLeadDocument{
		ID:           docID,
		LeadID:       leadID,
		DocumentType: incoming.Type,
		FilePath:     key,
		OriginalName: safeFilename(incoming.Filename),
		MimeType:     mimeType,
		SizeBytes:    size,
		CreatedAt:    now,
	}, nil
}

func normalizeSpecializations(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func hasIncomingDocument(docs []IncomingDocument, documentType DocumentType) bool {
	for _, doc := range docs {
		if doc.Type == documentType && doc.Reader != nil {
			return true
		}
	}
	return false
}

func closeIncomingDocuments(docs []IncomingDocument) {
	for _, doc := range docs {
		if doc.Reader != nil {
			_ = doc.Reader.Close()
		}
	}
}

func isValidDocumentType(documentType DocumentType) bool {
	switch documentType {
	case DocumentIdentity, DocumentEducation, DocumentIPRegistration, DocumentOther:
		return true
	default:
		return false
	}
}

func isValidStatus(status Status) bool {
	switch status {
	case StatusNew, StatusInReview, StatusApproved, StatusRejected, StatusConverted:
		return true
	default:
		return false
	}
}

func isValidManualStatus(status Status) bool {
	switch status {
	case StatusNew, StatusInReview, StatusApproved:
		return true
	default:
		return false
	}
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func safeFilename(filename string) string {
	name := strings.ReplaceAll(filename, "\\", "/")
	name = filepath.Base(name)
	if strings.TrimSpace(name) == "" || name == "." {
		return "document"
	}
	return name
}
