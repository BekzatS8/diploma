package courses

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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
	if err := tx.Commit(ctx); err != nil {
		return CourseAssignment{}, err
	}
	a.Progress = &CourseProgress{AssignmentID: a.ID, ExecutorID: a.ExecutorID, ProgressPercent: 0, Status: "assigned"}
	return a, nil
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
	if err != nil {
		return nil, 0, err
	}
	if err := r.attachProgress(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
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
		       c.id, c.coach_id, c.created_by, c.title, c.description, c.status, c.created_at, c.updated_at, c.deleted_at
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
		       c.id, c.coach_id, c.created_by, c.title, c.description, c.status, c.created_at, c.updated_at, c.deleted_at
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
		       c.id, c.coach_id, c.created_by, c.title, c.description, c.status, c.created_at, c.updated_at, c.deleted_at
		FROM course_assignments ca
		JOIN courses c ON c.id = ca.course_id
		WHERE ca.id=$1
	`, id)
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
