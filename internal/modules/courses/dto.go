package courses

import "time"

type CreateCourseRequest struct {
	Title              string   `json:"title" binding:"required,min=3,max=200"`
	Subtitle           *string  `json:"subtitle" binding:"omitempty,max=300"`
	Description        *string  `json:"description" binding:"omitempty,max=5000"`
	Slug               *string  `json:"slug" binding:"omitempty,max=180"`
	Category           *string  `json:"category" binding:"omitempty,max=120"`
	Level              *string  `json:"level" binding:"omitempty,oneof=beginner intermediate advanced"`
	Language           *string  `json:"language" binding:"omitempty,len=2"`
	Price              *float64 `json:"price" binding:"omitempty,gte=0"`
	Currency           *string  `json:"currency" binding:"omitempty,len=3"`
	DurationMinutes    *int     `json:"duration_minutes" binding:"omitempty,gte=0"`
	CoverUploadID      *string  `json:"cover_upload_id" binding:"omitempty,uuid"`
	CoverURL           *string  `json:"cover_url" binding:"omitempty,url,max=2048"`
	Tags               []string `json:"tags"`
	LearningOutcomes   []string `json:"learning_outcomes"`
	Requirements       []string `json:"requirements"`
	CertificateEnabled *bool    `json:"certificate_enabled"`
}

type UpdateCourseRequest struct {
	Title              *string  `json:"title" binding:"omitempty,min=3,max=200"`
	Subtitle           *string  `json:"subtitle" binding:"omitempty,max=300"`
	Description        *string  `json:"description" binding:"omitempty,max=5000"`
	Slug               *string  `json:"slug" binding:"omitempty,max=180"`
	Category           *string  `json:"category" binding:"omitempty,max=120"`
	Level              *string  `json:"level" binding:"omitempty,oneof=beginner intermediate advanced"`
	Language           *string  `json:"language" binding:"omitempty,len=2"`
	Price              *float64 `json:"price" binding:"omitempty,gte=0"`
	Currency           *string  `json:"currency" binding:"omitempty,len=3"`
	DurationMinutes    *int     `json:"duration_minutes" binding:"omitempty,gte=0"`
	CoverUploadID      *string  `json:"cover_upload_id" binding:"omitempty,uuid"`
	CoverURL           *string  `json:"cover_url" binding:"omitempty,url,max=2048"`
	Tags               []string `json:"tags"`
	LearningOutcomes   []string `json:"learning_outcomes"`
	Requirements       []string `json:"requirements"`
	CertificateEnabled *bool    `json:"certificate_enabled"`
}

type CreateMaterialRequest struct {
	Title           string         `json:"title" binding:"required,min=1,max=200"`
	Description     *string        `json:"description" binding:"omitempty,max=1000"`
	Type            string         `json:"type" binding:"required,oneof=video pdf link text"`
	UploadID        *string        `json:"upload_id" binding:"omitempty,uuid"`
	URL             *string        `json:"url" binding:"omitempty,url,max=2048"`
	Content         *string        `json:"content" binding:"omitempty,max=20000"`
	Position        *int           `json:"position" binding:"omitempty,gte=0"`
	DurationSeconds *int           `json:"duration_seconds" binding:"omitempty,gte=0"`
	IsPreview       *bool          `json:"is_preview"`
	Metadata        map[string]any `json:"metadata"`
}

type UpdateMaterialRequest struct {
	Title           *string        `json:"title" binding:"omitempty,min=1,max=200"`
	Description     *string        `json:"description" binding:"omitempty,max=1000"`
	Type            *string        `json:"type" binding:"omitempty,oneof=video pdf link text"`
	UploadID        *string        `json:"upload_id" binding:"omitempty,uuid"`
	URL             *string        `json:"url" binding:"omitempty,url,max=2048"`
	Content         *string        `json:"content" binding:"omitempty,max=20000"`
	Position        *int           `json:"position" binding:"omitempty,gte=0"`
	DurationSeconds *int           `json:"duration_seconds" binding:"omitempty,gte=0"`
	IsPreview       *bool          `json:"is_preview"`
	Metadata        map[string]any `json:"metadata"`
}

type ListCoursesQuery struct {
	Status   string
	Category string
	Search   string
	Page     int
	PageSize int
}

type CourseDetailResponse struct {
	Course    Course           `json:"course"`
	Materials []CourseMaterial `json:"materials"`
	Lessons   []CourseLesson   `json:"lessons"`
}

type CourseLesson struct {
	Key       string           `json:"key"`
	Title     string           `json:"title"`
	Position  int              `json:"position"`
	Materials []CourseMaterial `json:"materials"`
}

type CourseLearningResponse struct {
	Assignment CourseAssignment `json:"assignment"`
	Materials  []CourseMaterial `json:"materials"`
	Lessons    []CourseLesson   `json:"lessons"`
}

type CreateAssignmentRequest struct {
	ExecutorID string     `json:"executor_id" binding:"required,uuid"`
	CourseID   string     `json:"course_id" binding:"required,uuid"`
	Reason     *string    `json:"reason"`
	Source     string     `json:"source" binding:"required"`
	DueAt      *time.Time `json:"due_at"`
}

type EnrollCourseRequest struct {
	CourseID string `json:"course_id" binding:"required,uuid"`
}

type ListAssignmentsQuery struct {
	ExecutorID string
	CourseID   string
	Status     string
	Source     string
	Page       int
	PageSize   int
}

type CreatorAnalytics struct {
	TotalCourses           int     `json:"total_courses"`
	PublishedCourses       int     `json:"published_courses"`
	DraftCourses           int     `json:"draft_courses"`
	ArchivedCourses        int     `json:"archived_courses"`
	TotalMaterials         int     `json:"total_materials"`
	TotalAssignments       int     `json:"total_assignments"`
	ActiveStudents         int     `json:"active_students"`
	CompletedAssignments   int     `json:"completed_assignments"`
	AverageProgress        float64 `json:"average_progress"`
	ExecutorCanCreate      bool    `json:"executor_can_create"`
	ExecutorMinRating      float64 `json:"executor_min_rating,omitempty"`
	ExecutorMinReviewCount int     `json:"executor_min_review_count,omitempty"`
}

type CourseStudent struct {
	AssignmentID       string     `json:"assignment_id"`
	CourseID           string     `json:"course_id"`
	ExecutorID         string     `json:"executor_id"`
	ExecutorName       *string    `json:"executor_name,omitempty"`
	ExecutorEmail      *string    `json:"executor_email,omitempty"`
	Status             string     `json:"status"`
	ProgressPercent    int        `json:"progress_percent"`
	CompletedMaterials int        `json:"completed_materials"`
	TotalMaterials     int        `json:"total_materials"`
	AssignedAt         time.Time  `json:"assigned_at"`
	DueAt              *time.Time `json:"due_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	LastActivityAt     *time.Time `json:"last_activity_at,omitempty"`
}
