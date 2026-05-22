package courses

import "time"

type CreateCourseRequest struct {
	Title       string  `json:"title" binding:"required,min=3,max=200"`
	Description *string `json:"description" binding:"omitempty,max=5000"`
}

type UpdateCourseRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=3,max=200"`
	Description *string `json:"description" binding:"omitempty,max=5000"`
}

type CreateMaterialRequest struct {
	Title    string  `json:"title" binding:"required,min=1,max=200"`
	Type     string  `json:"type" binding:"required,oneof=video pdf link text"`
	UploadID *string `json:"upload_id" binding:"omitempty,uuid"`
	URL      *string `json:"url" binding:"omitempty,url,max=2048"`
	Content  *string `json:"content" binding:"omitempty,max=20000"`
	Position *int    `json:"position" binding:"omitempty,gte=0"`
}

type UpdateMaterialRequest struct {
	Title    *string `json:"title" binding:"omitempty,min=1,max=200"`
	Type     *string `json:"type" binding:"omitempty,oneof=video pdf link text"`
	UploadID *string `json:"upload_id" binding:"omitempty,uuid"`
	URL      *string `json:"url" binding:"omitempty,url,max=2048"`
	Content  *string `json:"content" binding:"omitempty,max=20000"`
	Position *int    `json:"position" binding:"omitempty,gte=0"`
}

type ListCoursesQuery struct {
	Status   string
	Page     int
	PageSize int
}

type CourseDetailResponse struct {
	Course    Course           `json:"course"`
	Materials []CourseMaterial `json:"materials"`
}

type CreateAssignmentRequest struct {
	ExecutorID string     `json:"executor_id" binding:"required,uuid"`
	CourseID   string     `json:"course_id" binding:"required,uuid"`
	Reason     *string    `json:"reason"`
	Source     string     `json:"source" binding:"required"`
	DueAt      *time.Time `json:"due_at"`
}

type ListAssignmentsQuery struct {
	ExecutorID string
	CourseID   string
	Status     string
	Source     string
	Page       int
	PageSize   int
}
