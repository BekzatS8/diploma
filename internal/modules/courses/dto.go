package courses

import "time"

type CreateCourseRequest struct {
	Title       string  `json:"title" binding:"required,min=3,max=200"`
	Description *string `json:"description"`
}

type UpdateCourseRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type CreateMaterialRequest struct {
	Title    string  `json:"title" binding:"required,min=1,max=200"`
	Type     string  `json:"type" binding:"required"`
	URL      *string `json:"url"`
	Content  *string `json:"content"`
	Position *int    `json:"position"`
}

type UpdateMaterialRequest struct {
	Title    *string `json:"title"`
	Type     *string `json:"type"`
	URL      *string `json:"url"`
	Content  *string `json:"content"`
	Position *int    `json:"position"`
}

type ListCoursesQuery struct {
	Status   string
	Page     int
	PageSize int
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
