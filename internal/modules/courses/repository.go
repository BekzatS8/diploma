package courses

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type CreateCourseParams struct {
	CreatorID          string
	Title              string
	Subtitle           *string
	Description        *string
	Slug               *string
	Category           *string
	Level              string
	Language           string
	Price              float64
	Currency           string
	DurationMinutes    int
	CoverUploadID      *string
	CoverURL           *string
	Tags               []string
	LearningOutcomes   []string
	Requirements       []string
	CertificateEnabled bool
}

type UpdateCourseParams struct {
	Title              *string
	Subtitle           *string
	Description        *string
	Slug               *string
	Category           *string
	Level              *string
	Language           *string
	Price              *float64
	Currency           *string
	DurationMinutes    *int
	CoverUploadID      *string
	CoverURL           *string
	Tags               []string
	LearningOutcomes   []string
	Requirements       []string
	CertificateEnabled *bool
}

type CreateMaterialParams struct {
	CourseID        string
	Title           string
	Description     *string
	MaterialType    string
	UploadID        *string
	URL             *string
	Content         *string
	SortOrder       int
	DurationSeconds int
	IsPreview       bool
	Metadata        map[string]any
}

type UpdateMaterialParams struct {
	Title           *string
	Description     *string
	MaterialType    *string
	UploadID        *string
	URL             *string
	Content         *string
	SortOrder       *int
	DurationSeconds *int
	IsPreview       *bool
	Metadata        map[string]any
}

type CreateAssignmentParams struct {
	CourseID   string
	ExecutorID string
	SanctionID *string
	AssignedBy string
	Reason     *string
	Source     string
	DueAt      interface{}
}

type scanner interface {
	Scan(dest ...any) error
}

func courseColumns(alias string) string {
	p := ""
	if alias != "" {
		p = alias + "."
	}
	return p + `id::text, ` + p + `coach_id::text, ` + p + `created_by::text, ` + p + `title, ` + p + `subtitle, ` + p + `description, ` +
		p + `slug, ` + p + `category, ` + p + `level, ` + p + `language, ` + p + `price::float8, ` + p + `currency, ` +
		p + `duration_minutes, ` + p + `cover_upload_id::text, ` + p + `cover_url, ` + p + `tags, ` + p + `learning_outcomes, ` +
		p + `requirements, ` + p + `certificate_enabled, ` + p + `status, ` + p + `moderation_status, ` + p + `enrollment_count, ` +
		p + `rating_avg::float8, ` + p + `rating_count, ` + p + `published_at, ` + p + `archived_at, ` + p + `created_at, ` + p + `updated_at, ` + p + `deleted_at`
}

func courseScanDest(c *Course) []any {
	return []any{
		&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Subtitle, &c.Description,
		&c.Slug, &c.Category, &c.Level, &c.Language, &c.Price, &c.Currency,
		&c.DurationMinutes, &c.CoverUploadID, &c.CoverURL, &c.Tags, &c.LearningOutcomes,
		&c.Requirements, &c.CertificateEnabled, &c.Status, &c.ModerationStatus, &c.EnrollmentCount,
		&c.RatingAvg, &c.RatingCount, &c.PublishedAt, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	}
}

func scanCourse(row scanner) (Course, error) {
	var c Course
	err := row.Scan(courseScanDest(&c)...)
	return c, err
}

func materialColumns() string {
	return `id::text, course_id::text, material_type, title, description, upload_id::text, url, content, sort_order, duration_seconds, is_preview, metadata, created_at, updated_at`
}

func scanMaterial(row scanner) (CourseMaterial, error) {
	var m CourseMaterial
	var metadata []byte
	err := row.Scan(&m.ID, &m.CourseID, &m.MaterialType, &m.Title, &m.Description, &m.UploadID, &m.URL, &m.Content, &m.SortOrder, &m.DurationSeconds, &m.IsPreview, &metadata, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return CourseMaterial{}, err
	}
	m.Metadata = map[string]any{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &m.Metadata); err != nil {
			return CourseMaterial{}, err
		}
	}
	return m, nil
}

func (r *Repository) CreateCourse(ctx context.Context, p CreateCourseParams) (Course, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO courses(
			coach_id, created_by, title, subtitle, description, slug, category, level, language,
			price, currency, duration_minutes, cover_upload_id, cover_url, tags, learning_outcomes,
			requirements, certificate_enabled, status, moderation_status
		)
		VALUES($1,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,'draft','approved')
		RETURNING `+courseColumns("")+`
	`, p.CreatorID, p.Title, p.Subtitle, p.Description, p.Slug, p.Category, p.Level, p.Language, p.Price, p.Currency, p.DurationMinutes, p.CoverUploadID, p.CoverURL, stringSliceOrEmpty(p.Tags), stringSliceOrEmpty(p.LearningOutcomes), stringSliceOrEmpty(p.Requirements), p.CertificateEnabled)
	return scanCourse(row)
}

func (r *Repository) UpdateCourse(ctx context.Context, id, ownerID string, isAdmin bool, p UpdateCourseParams) (Course, error) {
	query := `
		UPDATE courses SET
			title=COALESCE($3,title),
			subtitle=COALESCE($4,subtitle),
			description=COALESCE($5,description),
			slug=COALESCE($6,slug),
			category=COALESCE($7,category),
			level=COALESCE($8,level),
			language=COALESCE($9,language),
			price=COALESCE($10,price),
			currency=COALESCE($11,currency),
			duration_minutes=COALESCE($12,duration_minutes),
			cover_upload_id=COALESCE($13,cover_upload_id),
			cover_url=COALESCE($14,cover_url),
			tags=COALESCE($15,tags),
			learning_outcomes=COALESCE($16,learning_outcomes),
			requirements=COALESCE($17,requirements),
			certificate_enabled=COALESCE($18,certificate_enabled),
			updated_at=NOW()
		WHERE id=$1 AND deleted_at IS NULL`
	args := []interface{}{id, ownerID, p.Title, p.Subtitle, p.Description, p.Slug, p.Category, p.Level, p.Language, p.Price, p.Currency, p.DurationMinutes, p.CoverUploadID, p.CoverURL, nullableStringSlice(p.Tags), nullableStringSlice(p.LearningOutcomes), nullableStringSlice(p.Requirements), p.CertificateEnabled}
	if !isAdmin {
		query += ` AND (coach_id=$2 OR created_by=$2)`
	}
	query += ` RETURNING ` + courseColumns("")
	row := r.db.QueryRow(ctx, query, args...)
	return scanCourse(row)
}

func (r *Repository) GetCourseByID(ctx context.Context, id string) (Course, error) {
	row := r.db.QueryRow(ctx, `SELECT `+courseColumns("")+` FROM courses WHERE id=$1 AND deleted_at IS NULL`, id)
	return scanCourse(row)
}

func (r *Repository) ListCourses(ctx context.Context, role, userID string, q ListCoursesQuery) ([]Course, int64, error) {
	where := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argPos := 1
	if q.Status != "" {
		where = append(where, fmt.Sprintf("status=$%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	if q.Category != "" {
		where = append(where, fmt.Sprintf("category=$%d", argPos))
		args = append(args, q.Category)
		argPos++
	}
	if q.Search != "" {
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR subtitle ILIKE $%d OR description ILIKE $%d)", argPos, argPos, argPos))
		args = append(args, "%"+q.Search+"%")
		argPos++
	}
	if role == "coach" {
		where = append(where, fmt.Sprintf("(coach_id=$%d OR created_by=$%d)", argPos, argPos))
		args = append(args, userID)
		argPos++
	} else if role == "admin" {
		// admins can view all courses
	} else if role == "executor" {
		if q.Status == "" {
			where = append(where, "status='published'")
		}
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM courses WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, "SELECT "+courseColumns("")+" FROM courses WHERE "+whereSQL+" ORDER BY updated_at DESC, created_at DESC LIMIT $"+fmt.Sprintf("%d", argPos)+" OFFSET $"+fmt.Sprintf("%d", argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Course, 0)
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

func (r *Repository) TransitionCourseStatus(ctx context.Context, id, actorID, from, to string, isAdmin bool) (Course, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Course{}, err
	}
	defer tx.Rollback(ctx)
	query := `SELECT ` + courseColumns("") + ` FROM courses WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`
	row := tx.QueryRow(ctx, query, id)
	c, err := scanCourse(row)
	if err != nil {
		return Course{}, err
	}
	if c.Status != from {
		return Course{}, pgx.ErrNoRows
	}
	if !isAdmin {
		if (c.CoachID == nil || *c.CoachID != actorID) && (c.CreatedBy == nil || *c.CreatedBy != actorID) {
			return Course{}, pgx.ErrNoRows
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE courses
		SET status=$2,
			is_published=($2='published'),
			published_at=CASE WHEN $2='published' THEN COALESCE(published_at, NOW()) ELSE published_at END,
			archived_at=CASE WHEN $2='archived' THEN COALESCE(archived_at, NOW()) ELSE archived_at END,
			updated_at=NOW()
		WHERE id=$1
	`, id, to); err != nil {
		return Course{}, err
	}
	c, err = scanCourse(tx.QueryRow(ctx, `SELECT `+courseColumns("")+` FROM courses WHERE id=$1`, id))
	if err != nil {
		return Course{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Course{}, err
	}
	return c, nil
}

func (r *Repository) CreateMaterial(ctx context.Context, p CreateMaterialParams) (CourseMaterial, error) {
	metadata, err := json.Marshal(nonNilMap(p.Metadata))
	if err != nil {
		return CourseMaterial{}, err
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO course_materials(course_id, material_type, title, description, upload_id, url, content, sort_order, duration_seconds, is_preview, metadata)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING `+materialColumns()+`
	`, p.CourseID, p.MaterialType, p.Title, p.Description, p.UploadID, p.URL, p.Content, p.SortOrder, p.DurationSeconds, p.IsPreview, metadata)
	return scanMaterial(row)
}

func (r *Repository) UpdateMaterial(ctx context.Context, courseID, materialID string, p UpdateMaterialParams) (CourseMaterial, error) {
	var metadata any
	if p.Metadata != nil {
		value, err := json.Marshal(nonNilMap(p.Metadata))
		if err != nil {
			return CourseMaterial{}, err
		}
		metadata = value
	}
	row := r.db.QueryRow(ctx, `
		UPDATE course_materials SET
			title=COALESCE($3,title),
			description=COALESCE($4,description),
			material_type=COALESCE($5,material_type),
			upload_id=COALESCE($6,upload_id),
			url=COALESCE($7,url),
			content=COALESCE($8,content),
			sort_order=COALESCE($9,sort_order),
			duration_seconds=COALESCE($10,duration_seconds),
			is_preview=COALESCE($11,is_preview),
			metadata=COALESCE($12::jsonb,metadata),
			updated_at=NOW()
		WHERE id=$1 AND course_id=$2
		RETURNING `+materialColumns()+`
	`, materialID, courseID, p.Title, p.Description, p.MaterialType, p.UploadID, p.URL, p.Content, p.SortOrder, p.DurationSeconds, p.IsPreview, metadata)
	return scanMaterial(row)
}

func (r *Repository) DeleteMaterial(ctx context.Context, courseID, materialID string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM course_materials WHERE id=$1 AND course_id=$2`, materialID, courseID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListMaterialsByCourse(ctx context.Context, courseID string) ([]CourseMaterial, error) {
	rows, err := r.db.Query(ctx, `SELECT `+materialColumns()+` FROM course_materials WHERE course_id=$1 ORDER BY sort_order ASC, created_at ASC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CourseMaterial, 0)
	for rows.Next() {
		m, err := scanMaterial(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *Repository) CreateAssignment(ctx context.Context, p CreateAssignmentParams) (CourseAssignment, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CourseAssignment{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		INSERT INTO course_assignments(course_id, executor_id, sanction_id, assigned_by, reason, source, status, due_at)
		VALUES($1,$2,$3,$4,$5,$6,'assigned',$7)
		ON CONFLICT (course_id, executor_id) WHERE status IN ('assigned','in_progress') DO NOTHING
		RETURNING id, course_id, executor_id, sanction_id, assigned_by, reason, source, status, assigned_at, due_at, completed_at, created_at, updated_at
	`, p.CourseID, p.ExecutorID, p.SanctionID, p.AssignedBy, p.Reason, p.Source, p.DueAt)
	var a CourseAssignment
	if err := row.Scan(&a.ID, &a.CourseID, &a.ExecutorID, &a.SanctionID, &a.AssignedBy, &a.Reason, &a.Source, &a.Status, &a.AssignedAt, &a.DueAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return CourseAssignment{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO course_progress(assignment_id, executor_id, progress_percent, status)
		VALUES($1,$2,0,'assigned')
		ON CONFLICT (assignment_id) DO NOTHING
	`, a.ID, a.ExecutorID); err != nil {
		return CourseAssignment{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE courses
		SET enrollment_count=enrollment_count+1,
		    updated_at=NOW()
		WHERE id=$1
	`, a.CourseID); err != nil {
		return CourseAssignment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CourseAssignment{}, err
	}
	a.Progress = &CourseProgress{AssignmentID: a.ID, ExecutorID: a.ExecutorID, ProgressPercent: 0, Status: "assigned"}
	return a, nil
}

func (r *Repository) EnrollSelf(ctx context.Context, courseID, executorID string) (CourseAssignment, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CourseAssignment{}, false, err
	}
	defer tx.Rollback(ctx)

	existing, err := getMyAssignmentByCourseTx(ctx, tx, courseID, executorID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return CourseAssignment{}, false, err
		}
		items := []CourseAssignment{existing}
		if err := r.attachProgress(ctx, items); err != nil {
			return CourseAssignment{}, false, err
		}
		return items[0], false, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return CourseAssignment{}, false, err
	}

	var assignmentID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO course_assignments(course_id, executor_id, assigned_by, reason, source, status)
		VALUES($1,$2,$2,'Self-enrolled by executor','self_enrolled','assigned')
		RETURNING id::text
	`, courseID, executorID).Scan(&assignmentID); err != nil {
		return CourseAssignment{}, false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO course_progress(assignment_id, executor_id, progress_percent, status)
		VALUES($1,$2,0,'assigned')
		ON CONFLICT (assignment_id) DO NOTHING
	`, assignmentID, executorID); err != nil {
		return CourseAssignment{}, false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE courses
		SET enrollment_count=enrollment_count+1,
		    updated_at=NOW()
		WHERE id=$1
	`, courseID); err != nil {
		return CourseAssignment{}, false, err
	}
	item, err := getAssignmentWithCourseTx(ctx, tx, assignmentID)
	if err != nil {
		return CourseAssignment{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CourseAssignment{}, false, err
	}
	items := []CourseAssignment{item}
	if err := r.attachProgress(ctx, items); err != nil {
		return CourseAssignment{}, false, err
	}
	return items[0], true, nil
}

func (r *Repository) IsCoursePublished(ctx context.Context, courseID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM courses WHERE id=$1 AND deleted_at IS NULL AND status='published')`, courseID).Scan(&exists)
	return exists, err
}

func (r *Repository) ExecutorRating(ctx context.Context, executorID string) (float64, int, error) {
	var avg float64
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(rating)::float8, 5),
		       COUNT(*) FILTER (WHERE rating=5)::int
		FROM reviews
		WHERE reviewee_id=$1
		  AND reviewee_role='executor'
		  AND direction='client_to_executor'
		  AND deleted_at IS NULL
	`, executorID).Scan(&avg, &count)
	if err != nil {
		return 0, 0, err
	}
	return avg, count, nil
}

func (r *Repository) GetCreatorAnalytics(ctx context.Context, ownerID, role string, isAdmin bool) (CreatorAnalytics, error) {
	where := "c.deleted_at IS NULL"
	args := []any{}
	if !isAdmin {
		where += " AND (c.coach_id=$1 OR c.created_by=$1)"
		args = append(args, ownerID)
	}
	var out CreatorAnalytics
	if err := r.db.QueryRow(ctx, `
		WITH scoped_courses AS (
			SELECT c.*
			FROM courses c
			WHERE `+where+`
		),
		assignments AS (
			SELECT ca.*, cp.progress_percent
			FROM course_assignments ca
			JOIN scoped_courses c ON c.id = ca.course_id
			LEFT JOIN course_progress cp ON cp.assignment_id = ca.id
		)
		SELECT
			COALESCE((SELECT COUNT(*)::int FROM scoped_courses), 0),
			COALESCE((SELECT COUNT(*)::int FROM scoped_courses WHERE status='published'), 0),
			COALESCE((SELECT COUNT(*)::int FROM scoped_courses WHERE status='draft'), 0),
			COALESCE((SELECT COUNT(*)::int FROM scoped_courses WHERE status='archived'), 0),
			COALESCE((SELECT COUNT(*)::int FROM course_materials cm JOIN scoped_courses c ON c.id=cm.course_id), 0),
			COALESCE((SELECT COUNT(*)::int FROM assignments), 0),
			COALESCE((SELECT COUNT(DISTINCT executor_id)::int FROM assignments WHERE status IN ('assigned','in_progress')), 0),
			COALESCE((SELECT COUNT(*)::int FROM assignments WHERE status='completed'), 0),
			COALESCE((SELECT ROUND(AVG(COALESCE(progress_percent, 0))::numeric, 2)::float8 FROM assignments), 0)
	`, args...).Scan(
		&out.TotalCourses,
		&out.PublishedCourses,
		&out.DraftCourses,
		&out.ArchivedCourses,
		&out.TotalMaterials,
		&out.TotalAssignments,
		&out.ActiveStudents,
		&out.CompletedAssignments,
		&out.AverageProgress,
	); err != nil {
		return CreatorAnalytics{}, err
	}
	return out, nil
}

func (r *Repository) ListCourseStudents(ctx context.Context, courseID, ownerID string, isAdmin bool, page, size int) ([]CourseStudent, int64, error) {
	access := "c.id=$1 AND c.deleted_at IS NULL"
	args := []any{courseID}
	if !isAdmin {
		access += " AND (c.coach_id=$2 OR c.created_by=$2)"
		args = append(args, ownerID)
	}
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM course_assignments ca
		JOIN courses c ON c.id=ca.course_id
		WHERE `+access, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	limitPos := len(args) - 1
	offsetPos := len(args)
	rows, err := r.db.Query(ctx, `
		SELECT ca.id::text,
		       ca.course_id::text,
		       ca.executor_id::text,
		       ep.display_name,
		       u.email,
		       ca.status::text,
		       COALESCE(cp.progress_percent, 0),
		       COALESCE(done.completed_materials, 0),
		       COALESCE(total.total_materials, 0),
		       ca.assigned_at,
		       ca.due_at,
		       ca.completed_at,
		       cp.last_activity_at
		FROM course_assignments ca
		JOIN courses c ON c.id=ca.course_id
		JOIN users u ON u.id=ca.executor_id
		LEFT JOIN executor_profiles ep ON ep.user_id=ca.executor_id
		LEFT JOIN course_progress cp ON cp.assignment_id=ca.id
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS total_materials
			FROM course_materials cm
			WHERE cm.course_id=ca.course_id
		) total ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(cmp.material_id)::int AS completed_materials
			FROM course_material_progress cmp
			JOIN course_materials cm ON cm.id=cmp.material_id AND cm.course_id=ca.course_id
			WHERE cmp.assignment_id=ca.id
		) done ON TRUE
		WHERE `+access+`
		ORDER BY ca.assigned_at DESC
		LIMIT $`+fmt.Sprintf("%d", limitPos)+` OFFSET $`+fmt.Sprintf("%d", offsetPos), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]CourseStudent, 0)
	for rows.Next() {
		var item CourseStudent
		if err := rows.Scan(&item.AssignmentID, &item.CourseID, &item.ExecutorID, &item.ExecutorName, &item.ExecutorEmail, &item.Status, &item.ProgressPercent, &item.CompletedMaterials, &item.TotalMaterials, &item.AssignedAt, &item.DueAt, &item.CompletedAt, &item.LastActivityAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListAssignmentsAdmin(ctx context.Context, q ListAssignmentsQuery) ([]CourseAssignment, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argPos := 1
	if q.ExecutorID != "" {
		where = append(where, fmt.Sprintf("ca.executor_id=$%d", argPos))
		args = append(args, q.ExecutorID)
		argPos++
	}
	if q.CourseID != "" {
		where = append(where, fmt.Sprintf("ca.course_id=$%d", argPos))
		args = append(args, q.CourseID)
		argPos++
	}
	if q.Status != "" {
		where = append(where, fmt.Sprintf("ca.status=$%d", argPos))
		args = append(args, q.Status)
		argPos++
	}
	if q.Source != "" {
		where = append(where, fmt.Sprintf("ca.source=$%d", argPos))
		args = append(args, q.Source)
		argPos++
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM course_assignments ca WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       `+courseColumns("c")+`
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE `+whereSQL+`
		ORDER BY ca.assigned_at DESC
		LIMIT $`+fmt.Sprintf("%d", argPos)+` OFFSET $`+fmt.Sprintf("%d", argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanAssignmentsWithCourse(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.attachProgress(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) ListAssignmentsMy(ctx context.Context, executorID string, onlyPublished bool, status string, page, size int) ([]CourseAssignment, int64, error) {
	where := "ca.executor_id=$1"
	args := []any{executorID}
	argPos := 2
	if onlyPublished {
		where += " AND c.status='published' AND c.deleted_at IS NULL"
	}
	if status != "" {
		where += fmt.Sprintf(" AND ca.status=$%d", argPos)
		args = append(args, status)
		argPos++
	}
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM course_assignments ca JOIN courses c ON c.id=ca.course_id WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := r.db.Query(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       `+courseColumns("c")+`
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE `+where+`
		ORDER BY ca.assigned_at DESC
		LIMIT $`+fmt.Sprintf("%d", argPos)+` OFFSET $`+fmt.Sprintf("%d", argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanAssignmentsWithCourse(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.attachProgress(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) GetMyAssignmentByID(ctx context.Context, id, executorID string) (CourseAssignment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       `+courseColumns("c")+`
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE ca.id=$1 AND ca.executor_id=$2 AND c.status='published' AND c.deleted_at IS NULL
	`, id, executorID)
	item, err := scanAssignmentWithCourse(row)
	if err != nil {
		return CourseAssignment{}, err
	}
	items := []CourseAssignment{item}
	if err := r.attachProgress(ctx, items); err != nil {
		return CourseAssignment{}, err
	}
	return items[0], nil
}

func (r *Repository) GetMyAssignmentByCourseID(ctx context.Context, courseID, executorID string) (CourseAssignment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       `+courseColumns("c")+`
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE ca.course_id=$1
		  AND ca.executor_id=$2
		  AND ca.status <> 'cancelled'
		  AND c.status='published'
		  AND c.deleted_at IS NULL
		ORDER BY ca.assigned_at DESC
		LIMIT 1
	`, courseID, executorID)
	item, err := scanAssignmentWithCourse(row)
	if err != nil {
		return CourseAssignment{}, err
	}
	items := []CourseAssignment{item}
	if err := r.attachProgress(ctx, items); err != nil {
		return CourseAssignment{}, err
	}
	return items[0], nil
}

func (r *Repository) MarkAssignmentCompleted(ctx context.Context, id, executorID string) (CourseAssignment, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CourseAssignment{}, false, err
	}
	defer tx.Rollback(ctx)

	before, err := lockMyAssignmentTx(ctx, tx, id, executorID)
	if err != nil {
		return CourseAssignment{}, false, err
	}
	if before.Status == "cancelled" {
		return CourseAssignment{}, false, pgx.ErrNoRows
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO course_material_progress(assignment_id, material_id, executor_id)
		SELECT $1, cm.id, $2
		FROM course_materials cm
		WHERE cm.course_id=$3
		ON CONFLICT (assignment_id, material_id) DO UPDATE
		SET updated_at=NOW()
	`, before.ID, executorID, before.CourseID); err != nil {
		return CourseAssignment{}, false, err
	}
	progress, err := refreshAssignmentProgressTx(ctx, tx, before.ID, executorID, true)
	if err != nil {
		return CourseAssignment{}, false, err
	}
	after, err := getAssignmentWithCourseTx(ctx, tx, before.ID)
	if err != nil {
		return CourseAssignment{}, false, err
	}
	after.Progress = &progress
	completedNow := before.Status != "completed" && after.Status == "completed"
	if err := tx.Commit(ctx); err != nil {
		return CourseAssignment{}, false, err
	}
	return after, completedNow, nil
}

func (r *Repository) MarkMaterialCompleted(ctx context.Context, assignmentID, materialID, executorID string) (CourseAssignment, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CourseAssignment{}, false, err
	}
	defer tx.Rollback(ctx)

	before, err := lockMyAssignmentTx(ctx, tx, assignmentID, executorID)
	if err != nil {
		return CourseAssignment{}, false, err
	}
	if before.Status == "cancelled" {
		return CourseAssignment{}, false, pgx.ErrNoRows
	}

	var materialExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM course_materials
			WHERE id=$1 AND course_id=$2
		)
	`, materialID, before.CourseID).Scan(&materialExists); err != nil {
		return CourseAssignment{}, false, err
	}
	if !materialExists {
		return CourseAssignment{}, false, pgx.ErrNoRows
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO course_material_progress(assignment_id, material_id, executor_id)
		VALUES($1,$2,$3)
		ON CONFLICT (assignment_id, material_id) DO UPDATE
		SET updated_at=NOW()
	`, before.ID, materialID, executorID); err != nil {
		return CourseAssignment{}, false, err
	}

	progress, err := refreshAssignmentProgressTx(ctx, tx, before.ID, executorID, false)
	if err != nil {
		return CourseAssignment{}, false, err
	}
	after, err := getAssignmentWithCourseTx(ctx, tx, before.ID)
	if err != nil {
		return CourseAssignment{}, false, err
	}
	after.Progress = &progress
	completedNow := before.Status != "completed" && after.Status == "completed"
	if err := tx.Commit(ctx); err != nil {
		return CourseAssignment{}, false, err
	}
	return after, completedNow, nil
}

func lockMyAssignmentTx(ctx context.Context, tx pgx.Tx, id, executorID string) (CourseAssignment, error) {
	row := tx.QueryRow(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       `+courseColumns("c")+`
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE ca.id=$1 AND ca.executor_id=$2 AND c.status='published' AND c.deleted_at IS NULL
		FOR UPDATE OF ca
	`, id, executorID)
	return scanAssignmentWithCourse(row)
}

func getAssignmentWithCourseTx(ctx context.Context, tx pgx.Tx, id string) (CourseAssignment, error) {
	row := tx.QueryRow(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       `+courseColumns("c")+`
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE ca.id=$1
	`, id)
	return scanAssignmentWithCourse(row)
}

func getMyAssignmentByCourseTx(ctx context.Context, tx pgx.Tx, courseID, executorID string) (CourseAssignment, error) {
	row := tx.QueryRow(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       `+courseColumns("c")+`
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE ca.course_id=$1
		  AND ca.executor_id=$2
		  AND ca.status <> 'cancelled'
		  AND c.status='published'
		  AND c.deleted_at IS NULL
		ORDER BY ca.assigned_at DESC
		LIMIT 1
	`, courseID, executorID)
	return scanAssignmentWithCourse(row)
}

func refreshAssignmentProgressTx(ctx context.Context, tx pgx.Tx, assignmentID, executorID string, allowEmptyComplete bool) (CourseProgress, error) {
	var totalMaterials int
	var completedMaterials int
	var completedMaterialIDsCSV string
	if err := tx.QueryRow(ctx, `
		WITH assignment AS (
			SELECT course_id FROM course_assignments WHERE id=$1 AND executor_id=$2
		),
		total AS (
			SELECT COUNT(cm.id)::int AS total_materials
			FROM assignment a
			LEFT JOIN course_materials cm ON cm.course_id=a.course_id
		),
		completed AS (
			SELECT COUNT(cmp.material_id)::int AS completed_materials,
			       COALESCE(string_agg(cmp.material_id::text, ',' ORDER BY cmp.completed_at, cmp.material_id::text), '') AS material_ids
			FROM course_material_progress cmp
			JOIN course_materials cm ON cm.id=cmp.material_id
			JOIN assignment a ON a.course_id=cm.course_id
			WHERE cmp.assignment_id=$1
		)
		SELECT total.total_materials, completed.completed_materials, completed.material_ids
		FROM total CROSS JOIN completed
	`, assignmentID, executorID).Scan(&totalMaterials, &completedMaterials, &completedMaterialIDsCSV); err != nil {
		return CourseProgress{}, err
	}

	progressPercent := 0
	status := "assigned"
	if totalMaterials > 0 {
		progressPercent = completedMaterials * 100 / totalMaterials
		if completedMaterials >= totalMaterials {
			progressPercent = 100
			status = "completed"
		} else if completedMaterials > 0 {
			status = "in_progress"
		}
	} else if allowEmptyComplete {
		progressPercent = 100
		status = "completed"
	}

	if _, err := tx.Exec(ctx, `
		UPDATE course_assignments
		SET status=$2::course_assignment_status,
			completed_at=CASE WHEN $2='completed' THEN COALESCE(completed_at, NOW()) ELSE NULL END,
			updated_at=NOW()
		WHERE id=$1
	`, assignmentID, status); err != nil {
		return CourseProgress{}, err
	}

	var progress CourseProgress
	var createdAt time.Time
	var updatedAt time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO course_progress(assignment_id, executor_id, progress_percent, status, last_activity_at, completed_at)
		VALUES($1,$2,$3,$4::course_assignment_status,NOW(),CASE WHEN $4='completed' THEN NOW() ELSE NULL END)
		ON CONFLICT (assignment_id) DO UPDATE
		SET progress_percent=EXCLUDED.progress_percent,
			status=EXCLUDED.status,
			last_activity_at=NOW(),
			completed_at=CASE WHEN EXCLUDED.status::text='completed' THEN COALESCE(course_progress.completed_at, NOW()) ELSE NULL END,
			updated_at=NOW()
		RETURNING id, assignment_id, executor_id, progress_percent, status, last_activity_at, completed_at, created_at, updated_at
	`, assignmentID, executorID, progressPercent, status).Scan(&progress.ID, &progress.AssignmentID, &progress.ExecutorID, &progress.ProgressPercent, &progress.Status, &progress.LastActivityAt, &progress.CompletedAt, &createdAt, &updatedAt); err != nil {
		return CourseProgress{}, err
	}
	progress.CreatedAt = &createdAt
	progress.UpdatedAt = &updatedAt
	progress.TotalMaterials = totalMaterials
	progress.CompletedMaterials = completedMaterials
	progress.CompletedMaterialIDs = splitCSV(completedMaterialIDsCSV)
	return progress, nil
}

func (r *Repository) attachProgress(ctx context.Context, items []CourseAssignment) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	rows, err := r.db.Query(ctx, `
		SELECT ca.id::text, ca.executor_id::text,
		       cp.id::text, cp.progress_percent, cp.status::text, cp.last_activity_at, cp.completed_at, cp.created_at, cp.updated_at,
		       COALESCE(total.total_materials, 0), COALESCE(done.completed_materials, 0), COALESCE(done.material_ids, '')
		FROM course_assignments ca
		LEFT JOIN course_progress cp ON cp.assignment_id=ca.id
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS total_materials
			FROM course_materials cm
			WHERE cm.course_id=ca.course_id
		) total ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(cmp.material_id)::int AS completed_materials,
			       COALESCE(string_agg(cmp.material_id::text, ',' ORDER BY cmp.completed_at, cmp.material_id::text), '') AS material_ids
			FROM course_material_progress cmp
			JOIN course_materials cm ON cm.id=cmp.material_id AND cm.course_id=ca.course_id
			WHERE cmp.assignment_id=ca.id
		) done ON TRUE
		WHERE ca.id::text = ANY($1::text[])
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	byAssignment := make(map[string]*CourseProgress, len(items))
	for rows.Next() {
		var assignmentID string
		var executorID string
		var progressID sql.NullString
		var progressPercent sql.NullInt64
		var status sql.NullString
		var lastActivityAt sql.NullTime
		var completedAt sql.NullTime
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		var totalMaterials int
		var completedMaterials int
		var completedMaterialIDsCSV string
		if err := rows.Scan(&assignmentID, &executorID, &progressID, &progressPercent, &status, &lastActivityAt, &completedAt, &createdAt, &updatedAt, &totalMaterials, &completedMaterials, &completedMaterialIDsCSV); err != nil {
			return err
		}

		progress := &CourseProgress{
			AssignmentID:         assignmentID,
			ExecutorID:           executorID,
			Status:               "assigned",
			CompletedMaterials:   completedMaterials,
			TotalMaterials:       totalMaterials,
			CompletedMaterialIDs: splitCSV(completedMaterialIDsCSV),
		}
		if progressID.Valid {
			progress.ID = progressID.String
		}
		if progressPercent.Valid {
			progress.ProgressPercent = int(progressPercent.Int64)
		} else if totalMaterials > 0 {
			progress.ProgressPercent = completedMaterials * 100 / totalMaterials
		}
		if status.Valid {
			progress.Status = status.String
		}
		if lastActivityAt.Valid {
			progress.LastActivityAt = &lastActivityAt.Time
		}
		if completedAt.Valid {
			progress.CompletedAt = &completedAt.Time
		}
		if createdAt.Valid {
			progress.CreatedAt = &createdAt.Time
		}
		if updatedAt.Valid {
			progress.UpdatedAt = &updatedAt.Time
		}
		byAssignment[assignmentID] = progress
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range items {
		progress, ok := byAssignment[items[i].ID]
		if !ok {
			progress = &CourseProgress{
				AssignmentID: items[i].ID,
				ExecutorID:   items[i].ExecutorID,
				Status:       items[i].Status,
			}
		}
		items[i].Progress = progress
	}
	return nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

func nullableStringSlice(values []string) any {
	if values == nil {
		return nil
	}
	return values
}

func stringSliceOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func scanAssignmentsWithCourse(rows pgx.Rows) ([]CourseAssignment, error) {
	items := make([]CourseAssignment, 0)
	for rows.Next() {
		var a CourseAssignment
		var c Course
		a.Course = &c
		dest := []any{&a.ID, &a.CourseID, &a.ExecutorID, &a.SanctionID, &a.AssignedBy, &a.Reason, &a.Source, &a.Status, &a.AssignedAt, &a.DueAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt}
		dest = append(dest, courseScanDest(&c)...)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func scanAssignmentWithCourse(row pgx.Row) (CourseAssignment, error) {
	var a CourseAssignment
	var c Course
	a.Course = &c
	dest := []any{&a.ID, &a.CourseID, &a.ExecutorID, &a.SanctionID, &a.AssignedBy, &a.Reason, &a.Source, &a.Status, &a.AssignedAt, &a.DueAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt}
	dest = append(dest, courseScanDest(&c)...)
	err := row.Scan(dest...)
	return a, err
}
