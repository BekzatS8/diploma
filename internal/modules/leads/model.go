package leads

import "time"

type Status string

const (
	StatusNew       Status = "new"
	StatusInReview  Status = "in_review"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
	StatusConverted Status = "converted"
)

type DocumentType string

const (
	DocumentIdentity       DocumentType = "identity"
	DocumentEducation      DocumentType = "education"
	DocumentIPRegistration DocumentType = "ip_registration"
	DocumentOther          DocumentType = "other"
)

type ExecutorLead struct {
	ID              string
	Email           string
	PasswordHash    string
	FirstName       string
	LastName        string
	MiddleName      *string
	IIN             string
	Phone           string
	City            string
	ExperienceLevel string
	Specializations []string
	Education       string
	WorkFormat      *string
	HourlyRate      *float64
	About           string
	TermsAccepted   bool
	Status          Status
	Priority        int
	Notes           *string
	RejectionReason *string
	Source          *string
	UTMSource       *string
	UTMMedium       *string
	UTMCampaign     *string
	IPAddress       *string
	UserAgent       *string
	SubmittedAt     time.Time
	ReviewedAt      *time.Time
	ReviewedBy      *string
	ConvertedAt     *time.Time
	ConvertedUserID *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Documents       []ExecutorLeadDocument
}

type ExecutorLeadDocument struct {
	ID           string
	LeadID       string
	DocumentType DocumentType
	FilePath     string
	OriginalName string
	MimeType     string
	SizeBytes    int64
	CreatedAt    time.Time
	URL          string
}
