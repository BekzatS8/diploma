package courses

import "time"

type Course struct {
	ID          string     `json:"id"`
	CoachID     *string    `json:"coach_id,omitempty"`
	CreatedBy   *string    `json:"created_by,omitempty"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type CourseMaterial struct {
	ID           string     `json:"id"`
	CourseID     string     `json:"course_id"`
	MaterialType string     `json:"type"`
	Title        string     `json:"title"`
	UploadID     *string    `json:"upload_id,omitempty"`
	URL          *string    `json:"url,omitempty"`
	Content      *string    `json:"content,omitempty"`
	SortOrder    int        `json:"position"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
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
