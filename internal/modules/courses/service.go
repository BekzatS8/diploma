package courses

import (
	"context"
	"errors"
	"strings"

	notifications "buhpro/internal/modules/notifications"
	uploads "buhpro/internal/modules/uploads"

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
	repo                      *Repository
	notifier                  *notifications.Service
	uploads                   *uploads.Service
	executorCreatorMinRating  float64
	executorCreatorMinReviews int
}

type ServiceOptions struct {
	ExecutorCreatorMinRating  float64
	ExecutorCreatorMinReviews int
}

func NewService(repo *Repository, notifier *notifications.Service, uploads *uploads.Service, opts ServiceOptions) *Service {
	if opts.ExecutorCreatorMinRating <= 0 {
		opts.ExecutorCreatorMinRating = 5
	}
	if opts.ExecutorCreatorMinReviews < 0 {
		opts.ExecutorCreatorMinReviews = 0
	}
	return &Service{repo: repo, notifier: notifier, uploads: uploads, executorCreatorMinRating: opts.ExecutorCreatorMinRating, executorCreatorMinReviews: opts.ExecutorCreatorMinReviews}
}

func (s *Service) CreateCourse(ctx context.Context, userID, role string, req CreateCourseRequest) (Course, error) {
	allowed, err := s.canCreateCourse(ctx, userID, role)
	if err != nil {
		return Course{}, err
	}
	if !allowed {
		return Course{}, ErrForbidden
	}
	params, err := s.createCourseParams(ctx, userID, role, req)
	if err != nil {
		return Course{}, err
	}
	return s.repo.CreateCourse(ctx, params)
}

func (s *Service) UpdateCourse(ctx context.Context, id, userID, role string, req UpdateCourseRequest) (Course, error) {
	if !s.creatorRole(role) {
		return Course{}, ErrForbidden
	}
	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		if v == "" {
			return Course{}, ErrInvalidInput
		}
		req.Title = &v
	}
	params, err := s.updateCourseParams(ctx, userID, role, req)
	if err != nil {
		return Course{}, err
	}
	item, err := s.repo.UpdateCourse(ctx, id, userID, role == "admin", params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Course{}, ErrNotFound
		}
		return Course{}, err
	}
	return item, nil
}

func (s *Service) GetCoachCourse(ctx context.Context, id, userID, role string) (Course, []CourseMaterial, error) {
	if !s.creatorRole(role) {
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
	if !s.creatorRole(role) {
		return nil, 0, ErrForbidden
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	listRole := role
	if role == "executor" {
		listRole = "coach"
	}
	items, total, err := s.repo.ListCourses(ctx, listRole, userID, q)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) PublishCourse(ctx context.Context, id, userID, role string) (Course, error) {
	if !s.creatorRole(role) {
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
	if !s.creatorRole(role) {
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
	if !s.creatorRole(role) {
		return CourseMaterial{}, ErrForbidden
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return CourseMaterial{}, ErrInvalidInput
	}
	if _, _, err := s.GetCoachCourse(ctx, courseID, userID, role); err != nil {
		return CourseMaterial{}, err
	}
	uploadID, err := normalizeUUIDNullable(req.UploadID)
	if err != nil {
		return CourseMaterial{}, ErrInvalidInput
	}
	if err := s.ensureUploadAllowed(ctx, uploadID, userID, role); err != nil {
		return CourseMaterial{}, err
	}
	mt, err := validateMaterial(req.Type, uploadID, req.URL, req.Content)
	if err != nil {
		return CourseMaterial{}, err
	}
	position := 0
	if req.Position != nil {
		if *req.Position < 0 {
			return CourseMaterial{}, ErrInvalidInput
		}
		position = *req.Position
	}
	duration := 0
	if req.DurationSeconds != nil {
		duration = *req.DurationSeconds
	}
	isPreview := false
	if req.IsPreview != nil {
		isPreview = *req.IsPreview
	}
	item, err := s.repo.CreateMaterial(ctx, CreateMaterialParams{CourseID: courseID, Title: title, Description: normalizeNullable(req.Description), MaterialType: mt, UploadID: uploadID, URL: normalizeNullable(req.URL), Content: normalizeNullable(req.Content), SortOrder: position, DurationSeconds: duration, IsPreview: isPreview, Metadata: req.Metadata})
	if err != nil {
		return CourseMaterial{}, err
	}
	return item, nil
}

func (s *Service) UpdateMaterial(ctx context.Context, courseID, materialID, userID, role string, req UpdateMaterialRequest) (CourseMaterial, error) {
	if !s.creatorRole(role) {
		return CourseMaterial{}, ErrForbidden
	}
	if _, _, err := s.GetCoachCourse(ctx, courseID, userID, role); err != nil {
		return CourseMaterial{}, err
	}
	var materialType *string
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return CourseMaterial{}, ErrInvalidInput
	}
	if req.Position != nil && *req.Position < 0 {
		return CourseMaterial{}, ErrInvalidInput
	}
	uploadID, err := normalizeUUIDNullable(req.UploadID)
	if err != nil {
		return CourseMaterial{}, ErrInvalidInput
	}
	if err := s.ensureUploadAllowed(ctx, uploadID, userID, role); err != nil {
		return CourseMaterial{}, err
	}
	if req.Type != nil {
		mt, err := validateMaterial(*req.Type, uploadID, req.URL, req.Content)
		if err != nil {
			return CourseMaterial{}, err
		}
		materialType = &mt
	}
	item, err := s.repo.UpdateMaterial(ctx, courseID, materialID, UpdateMaterialParams{Title: normalizeNullable(req.Title), Description: normalizeNullable(req.Description), MaterialType: materialType, UploadID: uploadID, URL: normalizeNullable(req.URL), Content: normalizeNullable(req.Content), SortOrder: req.Position, DurationSeconds: req.DurationSeconds, IsPreview: req.IsPreview, Metadata: req.Metadata})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CourseMaterial{}, ErrNotFound
		}
		return CourseMaterial{}, err
	}
	return item, nil
}

func (s *Service) ensureUploadAllowed(ctx context.Context, uploadID *string, userID, role string) error {
	if uploadID == nil || role == "admin" {
		return nil
	}
	if s.uploads == nil {
		return ErrForbidden
	}
	item, err := s.uploads.GetByID(ctx, *uploadID)
	if err != nil {
		if errors.Is(err, uploads.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if item.AuthorID != userID {
		return ErrForbidden
	}
	return nil
}

func (s *Service) DeleteMaterial(ctx context.Context, courseID, materialID, userID, role string) error {
	if !s.creatorRole(role) {
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
	item, completedNow, err := s.repo.MarkAssignmentCompleted(ctx, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CourseAssignment{}, ErrNotFound
		}
		return CourseAssignment{}, err
	}
	if completedNow && s.notifier != nil && item.AssignedBy != nil && *item.AssignedBy != "" {
		_, _ = s.notifier.EmitInApp(ctx, *item.AssignedBy, notifications.TypeCourseCompleted, map[string]any{
			"course_id":            item.CourseID,
			"course_assignment_id": item.ID,
			"executor_id":          item.ExecutorID,
		})
	}
	return item, nil
}

func (s *Service) MarkMaterialCompleted(ctx context.Context, assignmentID, materialID, userID, role string) (CourseAssignment, error) {
	if role != "executor" {
		return CourseAssignment{}, ErrForbidden
	}
	item, completedNow, err := s.repo.MarkMaterialCompleted(ctx, assignmentID, materialID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CourseAssignment{}, ErrNotFound
		}
		return CourseAssignment{}, err
	}
	if completedNow && s.notifier != nil && item.AssignedBy != nil && *item.AssignedBy != "" {
		_, _ = s.notifier.EmitInApp(ctx, *item.AssignedBy, notifications.TypeCourseCompleted, map[string]any{
			"course_id":            item.CourseID,
			"course_assignment_id": item.ID,
			"executor_id":          item.ExecutorID,
		})
	}
	return item, nil
}

func (s *Service) CreatorAnalytics(ctx context.Context, userID, role string) (CreatorAnalytics, error) {
	if !s.creatorRole(role) {
		return CreatorAnalytics{}, ErrForbidden
	}
	item, err := s.repo.GetCreatorAnalytics(ctx, userID, role, role == "admin")
	if err != nil {
		return CreatorAnalytics{}, err
	}
	item.ExecutorMinRating = s.executorCreatorMinRating
	item.ExecutorMinReviewCount = s.executorCreatorMinReviews
	if role == "executor" {
		allowed, err := s.canCreateCourse(ctx, userID, role)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return CreatorAnalytics{}, err
		}
		item.ExecutorCanCreate = allowed
	} else {
		item.ExecutorCanCreate = true
	}
	return item, nil
}

func (s *Service) ListCourseStudents(ctx context.Context, courseID, userID, role string, page, pageSize int) ([]CourseStudent, int64, error) {
	if !s.creatorRole(role) {
		return nil, 0, ErrForbidden
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := s.repo.ListCourseStudents(ctx, courseID, userID, role == "admin", page, pageSize)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) creatorRole(role string) bool {
	return role == "coach" || role == "executor" || role == "admin"
}

func (s *Service) canCreateCourse(ctx context.Context, userID, role string) (bool, error) {
	switch role {
	case "coach", "admin":
		return true, nil
	case "executor":
		avg, count, err := s.repo.ExecutorRating(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, nil
			}
			return false, err
		}
		return avg >= s.executorCreatorMinRating && count >= s.executorCreatorMinReviews, nil
	default:
		return false, nil
	}
}

func (s *Service) createCourseParams(ctx context.Context, userID, role string, req CreateCourseRequest) (CreateCourseParams, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return CreateCourseParams{}, ErrInvalidInput
	}
	coverUploadID, err := normalizeUUIDNullable(req.CoverUploadID)
	if err != nil {
		return CreateCourseParams{}, ErrInvalidInput
	}
	if err := s.ensureUploadAllowed(ctx, coverUploadID, userID, role); err != nil {
		return CreateCourseParams{}, err
	}
	level := "beginner"
	if req.Level != nil {
		level = strings.ToLower(strings.TrimSpace(*req.Level))
	}
	language := "ru"
	if req.Language != nil {
		language = strings.ToLower(strings.TrimSpace(*req.Language))
	}
	price := 0.0
	if req.Price != nil {
		price = *req.Price
	}
	currency := "KZT"
	if req.Currency != nil {
		currency = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}
	duration := 0
	if req.DurationMinutes != nil {
		duration = *req.DurationMinutes
	}
	certificateEnabled := true
	if req.CertificateEnabled != nil {
		certificateEnabled = *req.CertificateEnabled
	}
	return CreateCourseParams{
		CreatorID:          userID,
		Title:              title,
		Subtitle:           normalizeNullable(req.Subtitle),
		Description:        normalizeNullable(req.Description),
		Slug:               normalizeSlug(req.Slug),
		Category:           normalizeNullable(req.Category),
		Level:              level,
		Language:           language,
		Price:              price,
		Currency:           currency,
		DurationMinutes:    duration,
		CoverUploadID:      coverUploadID,
		CoverURL:           normalizeNullable(req.CoverURL),
		Tags:               normalizeStringList(req.Tags),
		LearningOutcomes:   normalizeStringList(req.LearningOutcomes),
		Requirements:       normalizeStringList(req.Requirements),
		CertificateEnabled: certificateEnabled,
	}, nil
}

func (s *Service) updateCourseParams(ctx context.Context, userID, role string, req UpdateCourseRequest) (UpdateCourseParams, error) {
	coverUploadID, err := normalizeUUIDNullable(req.CoverUploadID)
	if err != nil {
		return UpdateCourseParams{}, ErrInvalidInput
	}
	if err := s.ensureUploadAllowed(ctx, coverUploadID, userID, role); err != nil {
		return UpdateCourseParams{}, err
	}
	var language *string
	if req.Language != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Language))
		language = &value
	}
	var currency *string
	if req.Currency != nil {
		value := strings.ToUpper(strings.TrimSpace(*req.Currency))
		currency = &value
	}
	var level *string
	if req.Level != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Level))
		level = &value
	}
	return UpdateCourseParams{
		Title:              normalizeNullable(req.Title),
		Subtitle:           normalizeNullable(req.Subtitle),
		Description:        normalizeNullable(req.Description),
		Slug:               normalizeSlug(req.Slug),
		Category:           normalizeNullable(req.Category),
		Level:              level,
		Language:           language,
		Price:              req.Price,
		Currency:           currency,
		DurationMinutes:    req.DurationMinutes,
		CoverUploadID:      coverUploadID,
		CoverURL:           normalizeNullable(req.CoverURL),
		Tags:               normalizeStringList(req.Tags),
		LearningOutcomes:   normalizeStringList(req.LearningOutcomes),
		Requirements:       normalizeStringList(req.Requirements),
		CertificateEnabled: req.CertificateEnabled,
	}, nil
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

func normalizeSlug(v *string) *string {
	value := normalizeNullable(v)
	if value == nil {
		return nil
	}
	slug := strings.ToLower(*value)
	slug = strings.ReplaceAll(slug, " ", "-")
	return &slug
}

func normalizeStringList(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
