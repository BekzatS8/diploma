package courses

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type CreateCourseParams struct {
	CreatorID   string
	Title       string
	Description *string
}

type UpdateCourseParams struct {
	Title       *string
	Description *string
}

type CreateMaterialParams struct {
	CourseID     string
	Title        string
	MaterialType string
	UploadID     *string
	URL          *string
	Content      *string
	SortOrder    int
}

type UpdateMaterialParams struct {
	Title        *string
	MaterialType *string
	UploadID     *string
	URL          *string
	Content      *string
	SortOrder    *int
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

func (r *Repository) CreateCourse(ctx context.Context, p CreateCourseParams) (Course, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO courses(coach_id, created_by, title, description, status)
		VALUES($1,$1,$2,$3,'draft')
		RETURNING id, coach_id, created_by, title, description, status, created_at, updated_at, deleted_at
	`, p.CreatorID, p.Title, p.Description)
	var c Course
	err := row.Scan(&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	return c, err
}

func (r *Repository) UpdateCourse(ctx context.Context, id, ownerID string, isAdmin bool, p UpdateCourseParams) (Course, error) {
	query := `
		UPDATE courses SET
			title=COALESCE($3,title),
			description=COALESCE($4,description),
			updated_at=NOW()
		WHERE id=$1 AND deleted_at IS NULL`
	args := []interface{}{id, ownerID, p.Title, p.Description}
	if !isAdmin {
		query += ` AND (coach_id=$2 OR created_by=$2)`
	}
	query += ` RETURNING id, coach_id, created_by, title, description, status, created_at, updated_at, deleted_at`
	row := r.db.QueryRow(ctx, query, args...)
	var c Course
	err := row.Scan(&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	return c, err
}

func (r *Repository) GetCourseByID(ctx context.Context, id string) (Course, error) {
	row := r.db.QueryRow(ctx, `SELECT id, coach_id, created_by, title, description, status, created_at, updated_at, deleted_at FROM courses WHERE id=$1`, id)
	var c Course
	err := row.Scan(&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	return c, err
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
	if role == "coach" {
		where = append(where, fmt.Sprintf("(coach_id=$%d OR created_by=$%d)", argPos, argPos))
		args = append(args, userID)
		argPos++
	} else if role == "executor" {
		where = append(where, "status='published'")
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM courses WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.Query(ctx, "SELECT id, coach_id, created_by, title, description, status, created_at, updated_at, deleted_at FROM courses WHERE "+whereSQL+" ORDER BY created_at DESC LIMIT $"+fmt.Sprintf("%d", argPos)+" OFFSET $"+fmt.Sprintf("%d", argPos+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Course, 0)
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
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
	query := `SELECT id, coach_id, created_by, title, description, status, created_at, updated_at, deleted_at FROM courses WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`
	row := tx.QueryRow(ctx, query, id)
	var c Course
	if err := row.Scan(&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
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
	if _, err := tx.Exec(ctx, `UPDATE courses SET status=$2, is_published=($2='published'), updated_at=NOW() WHERE id=$1`, id, to); err != nil {
		return Course{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT id, coach_id, created_by, title, description, status, created_at, updated_at, deleted_at FROM courses WHERE id=$1`, id).Scan(&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
		return Course{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Course{}, err
	}
	return c, nil
}

func (r *Repository) CreateMaterial(ctx context.Context, p CreateMaterialParams) (CourseMaterial, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO course_materials(course_id, material_type, title, upload_id, url, content, sort_order)
		VALUES($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, course_id, material_type, title, upload_id::text, url, content, sort_order, created_at, updated_at
	`, p.CourseID, p.MaterialType, p.Title, p.UploadID, p.URL, p.Content, p.SortOrder)
	var m CourseMaterial
	err := row.Scan(&m.ID, &m.CourseID, &m.MaterialType, &m.Title, &m.UploadID, &m.URL, &m.Content, &m.SortOrder, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (r *Repository) UpdateMaterial(ctx context.Context, courseID, materialID string, p UpdateMaterialParams) (CourseMaterial, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE course_materials SET
			title=COALESCE($3,title),
			material_type=COALESCE($4,material_type),
			upload_id=COALESCE($5,upload_id),
			url=COALESCE($6,url),
			content=COALESCE($7,content),
			sort_order=COALESCE($8,sort_order),
			updated_at=NOW()
		WHERE id=$1 AND course_id=$2
		RETURNING id, course_id, material_type, title, upload_id::text, url, content, sort_order, created_at, updated_at
	`, materialID, courseID, p.Title, p.MaterialType, p.UploadID, p.URL, p.Content, p.SortOrder)
	var m CourseMaterial
	err := row.Scan(&m.ID, &m.CourseID, &m.MaterialType, &m.Title, &m.UploadID, &m.URL, &m.Content, &m.SortOrder, &m.CreatedAt, &m.UpdatedAt)
	return m, err
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
	rows, err := r.db.Query(ctx, `SELECT id, course_id, material_type, title, upload_id::text, url, content, sort_order, created_at, updated_at FROM course_materials WHERE course_id=$1 ORDER BY sort_order ASC, created_at ASC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CourseMaterial, 0)
	for rows.Next() {
		var m CourseMaterial
		if err := rows.Scan(&m.ID, &m.CourseID, &m.MaterialType, &m.Title, &m.UploadID, &m.URL, &m.Content, &m.SortOrder, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *Repository) CreateAssignment(ctx context.Context, p CreateAssignmentParams) (CourseAssignment, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO course_assignments(course_id, executor_id, sanction_id, assigned_by, reason, source, status, due_at)
		VALUES($1,$2,$3,$4,$5,$6,'assigned',$7)
		ON CONFLICT (course_id, executor_id) WHERE status IN ('assigned','in_progress') DO NOTHING
		RETURNING id, course_id, executor_id, sanction_id, assigned_by, reason, source, status, assigned_at, due_at, completed_at, created_at, updated_at
	`, p.CourseID, p.ExecutorID, p.SanctionID, p.AssignedBy, p.Reason, p.Source, p.DueAt)
	var a CourseAssignment
	err := row.Scan(&a.ID, &a.CourseID, &a.ExecutorID, &a.SanctionID, &a.AssignedBy, &a.Reason, &a.Source, &a.Status, &a.AssignedAt, &a.DueAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *Repository) IsCoursePublished(ctx context.Context, courseID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM courses WHERE id=$1 AND deleted_at IS NULL AND status='published')`, courseID).Scan(&exists)
	return exists, err
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
		       c.id, c.coach_id, c.created_by, c.title, c.description, c.status, c.created_at, c.updated_at, c.deleted_at
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
	return items, total, err
}

func (r *Repository) ListAssignmentsMy(ctx context.Context, executorID string, onlyPublished bool, page, size int) ([]CourseAssignment, int64, error) {
	where := "ca.executor_id=$1"
	if onlyPublished {
		where += " AND c.status='published' AND c.deleted_at IS NULL"
	}
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM course_assignments ca JOIN courses c ON c.id=ca.course_id WHERE `+where, executorID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       c.id, c.coach_id, c.created_by, c.title, c.description, c.status, c.created_at, c.updated_at, c.deleted_at
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE `+where+`
		ORDER BY ca.assigned_at DESC
		LIMIT $2 OFFSET $3
	`, executorID, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanAssignmentsWithCourse(rows)
	return items, total, err
}

func (r *Repository) GetMyAssignmentByID(ctx context.Context, id, executorID string) (CourseAssignment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT ca.id, ca.course_id, ca.executor_id, ca.sanction_id, ca.assigned_by, ca.reason, ca.source, ca.status, ca.assigned_at, ca.due_at, ca.completed_at, ca.created_at, ca.updated_at,
		       c.id, c.coach_id, c.created_by, c.title, c.description, c.status, c.created_at, c.updated_at, c.deleted_at
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE ca.id=$1 AND ca.executor_id=$2 AND c.status='published' AND c.deleted_at IS NULL
	`, id, executorID)
	return scanAssignmentWithCourse(row)
}

func (r *Repository) MarkAssignmentCompleted(ctx context.Context, id, executorID string) (CourseAssignment, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CourseAssignment{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		UPDATE course_assignments
		SET status='completed', completed_at=COALESCE(completed_at, NOW()), updated_at=NOW()
		WHERE id=$1 AND executor_id=$2 AND status IN ('assigned','in_progress')
		RETURNING id, course_id, executor_id, sanction_id, assigned_by, reason, source, status, assigned_at, due_at, completed_at, created_at, updated_at
	`, id, executorID)
	var a CourseAssignment
	if err := row.Scan(&a.ID, &a.CourseID, &a.ExecutorID, &a.SanctionID, &a.AssignedBy, &a.Reason, &a.Source, &a.Status, &a.AssignedAt, &a.DueAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return CourseAssignment{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO course_progress(assignment_id, executor_id, progress_percent, status, last_activity_at, completed_at)
		VALUES($1,$2,100,'completed',NOW(),NOW())
		ON CONFLICT (assignment_id) DO UPDATE
		SET progress_percent=100, status='completed', last_activity_at=NOW(), completed_at=COALESCE(course_progress.completed_at, NOW()), updated_at=NOW()
	`, a.ID, executorID); err != nil {
		return CourseAssignment{}, err
	}
	var c Course
	if err := tx.QueryRow(ctx, `SELECT c.id, c.coach_id, c.created_by, c.title, c.description, c.status, c.created_at, c.updated_at, c.deleted_at FROM courses c WHERE c.id=$1`, a.CourseID).Scan(&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
		return CourseAssignment{}, err
	}
	a.Course = &c
	if err := tx.Commit(ctx); err != nil {
		return CourseAssignment{}, err
	}
	return a, nil
}

func scanAssignmentsWithCourse(rows pgx.Rows) ([]CourseAssignment, error) {
	items := make([]CourseAssignment, 0)
	for rows.Next() {
		var a CourseAssignment
		var c Course
		a.Course = &c
		if err := rows.Scan(&a.ID, &a.CourseID, &a.ExecutorID, &a.SanctionID, &a.AssignedBy, &a.Reason, &a.Source, &a.Status, &a.AssignedAt, &a.DueAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt,
			&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
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
	err := row.Scan(&a.ID, &a.CourseID, &a.ExecutorID, &a.SanctionID, &a.AssignedBy, &a.Reason, &a.Source, &a.Status, &a.AssignedAt, &a.DueAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt,
		&c.ID, &c.CoachID, &c.CreatedBy, &c.Title, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	return a, err
}
