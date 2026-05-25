package courses

import "time"

type Course struct {
	ID                 string     `json:"id"`
	CoachID            *string    `json:"coach_id,omitempty"`
	CreatedBy          *string    `json:"created_by,omitempty"`
	Title              string     `json:"title"`
	Subtitle           *string    `json:"subtitle,omitempty"`
	Description        *string    `json:"description,omitempty"`
	Slug               *string    `json:"slug,omitempty"`
	Category           *string    `json:"category,omitempty"`
	Level              string     `json:"level"`
	Language           string     `json:"language"`
	Price              float64    `json:"price"`
	Currency           string     `json:"currency"`
	DurationMinutes    int        `json:"duration_minutes"`
	CoverUploadID      *string    `json:"cover_upload_id,omitempty"`
	CoverURL           *string    `json:"cover_url,omitempty"`
	Tags               []string   `json:"tags"`
	LearningOutcomes   []string   `json:"learning_outcomes"`
	Requirements       []string   `json:"requirements"`
	CertificateEnabled bool       `json:"certificate_enabled"`
	Status             string     `json:"status"`
	ModerationStatus   string     `json:"moderation_status"`
	EnrollmentCount    int        `json:"enrollment_count"`
	RatingAvg          float64    `json:"rating_avg"`
	RatingCount        int        `json:"rating_count"`
	PublishedAt        *time.Time `json:"published_at,omitempty"`
	ArchivedAt         *time.Time `json:"archived_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type CourseMaterial struct {
	ID              string         `json:"id"`
	CourseID        string         `json:"course_id"`
	MaterialType    string         `json:"type"`
	Title           string         `json:"title"`
	Description     *string        `json:"description,omitempty"`
	UploadID        *string        `json:"upload_id,omitempty"`
	URL             *string        `json:"url,omitempty"`
	Content         *string        `json:"content,omitempty"`
	SortOrder       int            `json:"position"`
	DurationSeconds int            `json:"duration_seconds"`
	IsPreview       bool           `json:"is_preview"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       *time.Time     `json:"updated_at,omitempty"`
}

type CourseAssignment struct {
	ID          string          `json:"id"`
	CourseID    string          `json:"course_id"`
	ExecutorID  string          `json:"executor_id"`
	SanctionID  *string         `json:"sanction_id,omitempty"`
	AssignedBy  *string         `json:"assigned_by,omitempty"`
	Reason      *string         `json:"reason,omitempty"`
	Source      string          `json:"source"`
	Status      string          `json:"status"`
	AssignedAt  time.Time       `json:"assigned_at"`
	DueAt       *time.Time      `json:"due_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Course      *Course         `json:"course,omitempty"`
	Progress    *CourseProgress `json:"progress,omitempty"`
}

type CourseProgress struct {
	ID                   string     `json:"id,omitempty"`
	AssignmentID         string     `json:"assignment_id"`
	ExecutorID           string     `json:"executor_id"`
	ProgressPercent      int        `json:"progress_percent"`
	Status               string     `json:"status"`
	CompletedMaterials   int        `json:"completed_materials"`
	TotalMaterials       int        `json:"total_materials"`
	CompletedMaterialIDs []string   `json:"completed_material_ids,omitempty"`
	LastActivityAt       *time.Time `json:"last_activity_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	CreatedAt            *time.Time `json:"created_at,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

type CourseMaterialProgress struct {
	ID           string    `json:"id"`
	AssignmentID string    `json:"assignment_id"`
	MaterialID   string    `json:"material_id"`
	ExecutorID   string    `json:"executor_id"`
	CompletedAt  time.Time `json:"completed_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
