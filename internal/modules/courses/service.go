package courses

import (
	"context"
	"errors"
	"strings"

	notifications "buhpro/internal/modules/notifications"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
)

type Service struct {
	repo     *Repository
	notifier *notifications.Service
}

func NewService(repo *Repository, notifier *notifications.Service) *Service {
	return &Service{repo: repo, notifier: notifier}
}

func (s *Service) CreateCourse(ctx context.Context, userID, role string, req CreateCourseRequest) (Course, error) {
	if role != "coach" && role != "admin" {
		return Course{}, ErrForbidden
	}
	desc := normalizeNullable(req.Description)
	return s.repo.CreateCourse(ctx, CreateCourseParams{CreatorID: userID, Title: strings.TrimSpace(req.Title), Description: desc})
}

func (s *Service) UpdateCourse(ctx context.Context, id, userID, role string, req UpdateCourseRequest) (Course, error) {
	if role != "coach" && role != "admin" {
		return Course{}, ErrForbidden
	}
	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		if v == "" {
			return Course{}, ErrInvalidInput
		}
		req.Title = &v
	}
	req.Description = normalizeNullable(req.Description)
	item, err := s.repo.UpdateCourse(ctx, id, userID, role == "admin", UpdateCourseParams{Title: req.Title, Description: req.Description})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Course{}, ErrNotFound
		}
		return Course{}, err
	}
	return item, nil
}

func (s *Service) GetCoachCourse(ctx context.Context, id, userID, role string) (Course, []CourseMaterial, error) {
	if role != "coach" && role != "admin" {
		return Course{}, nil, ErrForbidden
	}
	item, err := s.repo.GetCourseByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Course{}, nil, ErrNotFound
		}
		return Course{}, nil, err
	}
	if role != "admin" && ((item.CoachID == nil || *item.CoachID != userID) && (item.CreatedBy == nil || *item.CreatedBy != userID)) {
		return Course{}, nil, ErrForbidden
	}
	materials, err := s.repo.ListMaterialsByCourse(ctx, item.ID)
	if err != nil {
		return Course{}, nil, err
	}
	return item, materials, nil
}

func (s *Service) ListCoachCourses(ctx context.Context, userID, role string, q ListCoursesQuery) ([]Course, int64, error) {
	if role != "coach" && role != "admin" {
		return nil, 0, ErrForbidden
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	items, total, err := s.repo.ListCourses(ctx, role, userID, q)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) PublishCourse(ctx context.Context, id, userID, role string) (Course, error) {
	if role != "coach" && role != "admin" {
		return Course{}, ErrForbidden
	}
	item, err := s.repo.TransitionCourseStatus(ctx, id, userID, "draft", "published", role == "admin")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Course{}, ErrConflict
		}
		return Course{}, err
	}
	return item, nil
}

func (s *Service) ArchiveCourse(ctx context.Context, id, userID, role string) (Course, error) {
	if role != "coach" && role != "admin" {
		return Course{}, ErrForbidden
	}
	item, err := s.repo.TransitionCourseStatus(ctx, id, userID, "published", "archived", role == "admin")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Course{}, ErrConflict
		}
		return Course{}, err
	}
	return item, nil
}

func (s *Service) AddMaterial(ctx context.Context, courseID, userID, role string, req CreateMaterialRequest) (CourseMaterial, error) {
	if role != "coach" && role != "admin" {
		return CourseMaterial{}, ErrForbidden
	}
	if _, _, err := s.GetCoachCourse(ctx, courseID, userID, role); err != nil {
		return CourseMaterial{}, err
	}
	uploadID, err := normalizeUUIDNullable(req.UploadID)
	if err != nil {
		return CourseMaterial{}, ErrInvalidInput
	}
	mt, err := validateMaterial(req.Type, uploadID, req.URL, req.Content)
	if err != nil {
		return CourseMaterial{}, err
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	item, err := s.repo.CreateMaterial(ctx, CreateMaterialParams{CourseID: courseID, Title: strings.TrimSpace(req.Title), MaterialType: mt, UploadID: uploadID, URL: normalizeNullable(req.URL), Content: normalizeNullable(req.Content), SortOrder: position})
	if err != nil {
		return CourseMaterial{}, err
	}
	return item, nil
}

func (s *Service) UpdateMaterial(ctx context.Context, courseID, materialID, userID, role string, req UpdateMaterialRequest) (CourseMaterial, error) {
	if role != "coach" && role != "admin" {
		return CourseMaterial{}, ErrForbidden
	}
	if _, _, err := s.GetCoachCourse(ctx, courseID, userID, role); err != nil {
		return CourseMaterial{}, err
	}
	var materialType *string
	uploadID, err := normalizeUUIDNullable(req.UploadID)
	if err != nil {
		return CourseMaterial{}, ErrInvalidInput
	}
	if req.Type != nil {
		mt, err := validateMaterial(*req.Type, uploadID, req.URL, req.Content)
		if err != nil {
			return CourseMaterial{}, err
		}
		materialType = &mt
	}
	item, err := s.repo.UpdateMaterial(ctx, courseID, materialID, UpdateMaterialParams{Title: normalizeNullable(req.Title), MaterialType: materialType, UploadID: uploadID, URL: normalizeNullable(req.URL), Content: normalizeNullable(req.Content), SortOrder: req.Position})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CourseMaterial{}, ErrNotFound
		}
		return CourseMaterial{}, err
	}
	return item, nil
}

func (s *Service) DeleteMaterial(ctx context.Context, courseID, materialID, userID, role string) error {
	if role != "coach" && role != "admin" {
		return ErrForbidden
	}
	if _, _, err := s.GetCoachCourse(ctx, courseID, userID, role); err != nil {
		return err
	}
	if err := s.repo.DeleteMaterial(ctx, courseID, materialID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ListCoursesForRole(ctx context.Context, userID, role string, q ListCoursesQuery) ([]Course, int64, error) {
	if role != "executor" && role != "coach" && role != "admin" {
		return nil, 0, ErrForbidden
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	if role == "executor" {
		q.Status = "published"
	}
	return s.repo.ListCourses(ctx, role, userID, q)
}

func (s *Service) GetCourseCatalogByID(ctx context.Context, id, userID, role string) (Course, []CourseMaterial, error) {
	if role != "executor" && role != "coach" && role != "admin" {
		return Course{}, nil, ErrForbidden
	}
	item, err := s.repo.GetCourseByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Course{}, nil, ErrNotFound
		}
		return Course{}, nil, err
	}
	if role == "executor" && item.Status != "published" {
		return Course{}, nil, ErrNotFound
	}
	materials, err := s.repo.ListMaterialsByCourse(ctx, item.ID)
	if err != nil {
		return Course{}, nil, err
	}
	return item, materials, nil
}

func (s *Service) CreateAssignment(ctx context.Context, userID, role string, req CreateAssignmentRequest) (CourseAssignment, error) {
	if role != "admin" {
		return CourseAssignment{}, ErrForbidden
	}
	source := strings.TrimSpace(req.Source)
	if source != "manual_admin" && source != "sanction_low_rating_first" && source != "sanction_low_rating_repeat" {
		return CourseAssignment{}, ErrInvalidInput
	}
	exists, err := s.repo.IsCoursePublished(ctx, req.CourseID)
	if err != nil {
		return CourseAssignment{}, err
	}
	if !exists {
		return CourseAssignment{}, ErrInvalidInput
	}
	item, err := s.repo.CreateAssignment(ctx, CreateAssignmentParams{CourseID: req.CourseID, ExecutorID: req.ExecutorID, AssignedBy: userID, Reason: normalizeNullable(req.Reason), Source: source, DueAt: req.DueAt})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CourseAssignment{}, ErrConflict
		}
		return CourseAssignment{}, err
	}
	if s.notifier != nil {
		_, _ = s.notifier.EmitInApp(ctx, req.ExecutorID, notifications.TypeCourseAssigned, map[string]any{
			"course_id":            item.CourseID,
			"course_assignment_id": item.ID,
			"sanction_id":          item.SanctionID,
			"source":               source,
		})
	}
	return item, nil
}

func (s *Service) ListAdminAssignments(ctx context.Context, role string, q ListAssignmentsQuery) ([]CourseAssignment, int64, error) {
	if role != "admin" {
		return nil, 0, ErrForbidden
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	return s.repo.ListAssignmentsAdmin(ctx, q)
}

func (s *Service) ListMyAssignments(ctx context.Context, userID, role string, page, pageSize int) ([]CourseAssignment, int64, error) {
	if role != "executor" {
		return nil, 0, ErrForbidden
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListAssignmentsMy(ctx, userID, true, page, pageSize)
}

func (s *Service) GetMyAssignmentByID(ctx context.Context, id, userID, role string) (CourseAssignment, error) {
	if role != "executor" {
		return CourseAssignment{}, ErrForbidden
	}
	item, err := s.repo.GetMyAssignmentByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CourseAssignment{}, ErrNotFound
		}
		return CourseAssignment{}, err
	}
	return item, nil
}

func (s *Service) MarkCompleted(ctx context.Context, id, userID, role string) (CourseAssignment, error) {
	if role != "executor" {
		return CourseAssignment{}, ErrForbidden
	}
	item, err := s.repo.MarkAssignmentCompleted(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CourseAssignment{}, ErrNotFound
		}
		return CourseAssignment{}, err
	}
	if s.notifier != nil && item.AssignedBy != nil && *item.AssignedBy != "" {
		_, _ = s.notifier.EmitInApp(ctx, *item.AssignedBy, notifications.TypeCourseCompleted, map[string]any{
			"course_id":            item.CourseID,
			"course_assignment_id": item.ID,
			"executor_id":          item.ExecutorID,
		})
	}
	return item, nil
}

func normalizeNullable(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	return &t
}

func normalizeUUIDNullable(v *string) (*string, error) {
	value := normalizeNullable(v)
	if value == nil {
		return nil, nil
	}
	if _, err := uuid.Parse(*value); err != nil {
		return nil, err
	}
	return value, nil
}

func validateMaterial(materialType string, uploadID, url, content *string) (string, error) {
	t := strings.TrimSpace(strings.ToLower(materialType))
	switch t {
	case "video", "pdf", "link", "text":
	default:
		return "", ErrInvalidInput
	}
	if (t == "video" || t == "pdf" || t == "link") && uploadID == nil && normalizeNullable(url) == nil {
		return "", ErrInvalidInput
	}
	if t == "text" && normalizeNullable(content) == nil {
		return "", ErrInvalidInput
	}
	return t, nil
}
