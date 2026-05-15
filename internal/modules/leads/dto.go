package leads

import (
	"io"
	"time"
)

type SubmitExecutorLeadRequest struct {
	Email           string
	Password        string
	FirstName       string
	LastName        string
	MiddleName      string
	IIN             string
	Phone           string
	City            string
	ExperienceLevel string
	Specializations []string
	Education       string
	WorkFormat      string
	HourlyRate      *float64
	About           string
	TermsAccepted   bool
	Source          string
	UTMSource       string
	UTMMedium       string
	UTMCampaign     string
	IPAddress       string
	UserAgent       string
	Documents       []IncomingDocument
}

type IncomingDocument struct {
	Type     DocumentType
	Filename string
	Reader   io.ReadCloser
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Notes  string `json:"notes"`
}

type RejectRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type ApproveRequest struct {
	Notes string `json:"notes"`
}

type ExecutorLeadSubmittedResponse struct {
	LeadID  string `json:"lead_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ExecutorLeadListResponse struct {
	Items    []ExecutorLeadView `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}

type ExecutorLeadView struct {
	ID              string                     `json:"id"`
	Email           string                     `json:"email"`
	FirstName       string                     `json:"first_name"`
	LastName        string                     `json:"last_name"`
	MiddleName      *string                    `json:"middle_name,omitempty"`
	IIN             string                     `json:"iin"`
	Phone           string                     `json:"phone"`
	City            string                     `json:"city"`
	ExperienceLevel string                     `json:"experience_level"`
	Specializations []string                   `json:"specializations"`
	Education       string                     `json:"education"`
	WorkFormat      *string                    `json:"work_format,omitempty"`
	HourlyRate      *float64                   `json:"hourly_rate,omitempty"`
	About           string                     `json:"about"`
	TermsAccepted   bool                       `json:"terms_accepted"`
	Status          string                     `json:"status"`
	Priority        int                        `json:"priority"`
	Notes           *string                    `json:"notes,omitempty"`
	RejectionReason *string                    `json:"rejection_reason,omitempty"`
	SubmittedAt     time.Time                  `json:"submitted_at"`
	ReviewedAt      *time.Time                 `json:"reviewed_at,omitempty"`
	ReviewedBy      *string                    `json:"reviewed_by,omitempty"`
	ConvertedAt     *time.Time                 `json:"converted_at,omitempty"`
	ConvertedUserID *string                    `json:"converted_user_id,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	Documents       []ExecutorLeadDocumentView `json:"documents,omitempty"`
}

type ExecutorLeadDocumentView struct {
	ID           string    `json:"id"`
	DocumentType string    `json:"document_type"`
	URL          string    `json:"url"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

type ApproveResponse struct {
	Status string `json:"status"`
	UserID string `json:"user_id"`
}
