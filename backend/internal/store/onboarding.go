package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"phystech-answer/backend/internal/model"
)

// ---------- documents: folders & visibility ----------

// UpdateDocumentAdmin changes the folder (category), visibility or title.
// nil fields are left unchanged.
func (p *Postgres) UpdateDocumentAdmin(ctx context.Context, id string, title, folder, visibility *string) error {
	res, err := p.pool.Exec(ctx, `UPDATE documents SET
		title      = COALESCE($2, title),
		category   = COALESCE($3, category),
		visibility = COALESCE($4, visibility),
		updated_at = now()
		WHERE id = $1`, id, title, folder, visibility)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) SetDocumentFolder(ctx context.Context, id, folder string) error {
	_, err := p.pool.Exec(ctx, `UPDATE documents SET category=$2 WHERE id=$1`, id, folder)
	return err
}

// ---------- departments ----------

func (p *Postgres) ListDepartments(ctx context.Context) ([]model.Department, error) {
	rows, err := p.pool.Query(ctx, `SELECT d.id, d.name, d.created_at,
		(SELECT count(*) FROM users u WHERE u.department_id = d.id AND u.active)
		FROM departments d ORDER BY d.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Department{}
	for rows.Next() {
		var d model.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.Members); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (p *Postgres) CreateDepartment(ctx context.Context, name string) (model.Department, error) {
	d := model.Department{ID: newID("dep"), Name: strings.TrimSpace(name), CreatedAt: time.Now()}
	_, err := p.pool.Exec(ctx, `INSERT INTO departments (id, name, created_at) VALUES ($1,$2,$3)`, d.ID, d.Name, d.CreatedAt)
	if isUniqueViolation(err) {
		return model.Department{}, ErrConflict
	}
	return d, err
}

func (p *Postgres) RenameDepartment(ctx context.Context, id, name string) error {
	res, err := p.pool.Exec(ctx, `UPDATE departments SET name=$2 WHERE id=$1`, id, strings.TrimSpace(name))
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if err == nil && res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (p *Postgres) DeleteDepartment(ctx context.Context, id string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM document_assignments WHERE target_type='department' AND target_id=$1`, id); err != nil {
		return err
	}
	res, err := tx.Exec(ctx, `DELETE FROM departments WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// ---------- users: organisation ----------

const profileSelect = `SELECT u.id, u.name, u.email, COALESCE(u.login, ''), u.role, u.created_at,
	COALESCE(u.department_id, ''), COALESCE(d.name, ''), u.position, u.active
	FROM users u LEFT JOIN departments d ON d.id = u.department_id`

func scanProfile(row pgx.Row) (model.UserProfile, error) {
	var u model.UserProfile
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Login, &u.Role, &u.CreatedAt,
		&u.DepartmentID, &u.DepartmentName, &u.Position, &u.Active)
	return u, err
}

func (p *Postgres) ListUserProfiles(ctx context.Context) ([]model.UserProfile, error) {
	rows, err := p.pool.Query(ctx, profileSelect+` ORDER BY u.active DESC, u.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.UserProfile{}
	for rows.Next() {
		u, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (p *Postgres) GetUserProfile(ctx context.Context, id string) (model.UserProfile, error) {
	u, err := scanProfile(p.pool.QueryRow(ctx, profileSelect+` WHERE u.id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.UserProfile{}, ErrNotFound
	}
	return u, err
}

// UpdateUserOrg changes role, department ("" clears it), position or active
// flag; nil fields are left unchanged. Deactivation ends the user's sessions.
func (p *Postgres) UpdateUserOrg(ctx context.Context, id string, role, departmentID, position *string, active *bool) error {
	var dep any
	clearDep := false
	if departmentID != nil {
		if *departmentID == "" {
			clearDep = true
		} else {
			dep = *departmentID
		}
	}
	res, err := p.pool.Exec(ctx, `UPDATE users SET
		role          = COALESCE($2, role),
		department_id = CASE WHEN $6 THEN NULL ELSE COALESCE($3::text, department_id) END,
		position      = COALESCE($4, position),
		active        = COALESCE($5, active)
		WHERE id = $1`, id, role, dep, position, active, clearDep)
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23503" {
			return ErrNotFound
		}
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	if active != nil && !*active {
		_, _ = p.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, id)
	}
	return nil
}

func (p *Postgres) IsUserActive(ctx context.Context, id string) (bool, error) {
	var a bool
	err := p.pool.QueryRow(ctx, `SELECT active FROM users WHERE id=$1`, id).Scan(&a)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return a, err
}

// ---------- assignments ----------

const assignmentSelect = `SELECT a.id, a.document_id, a.target_type, a.target_id,
	COALESCE(CASE a.target_type
		WHEN 'department' THEN (SELECT name FROM departments WHERE id = a.target_id)
		WHEN 'user' THEN (SELECT name FROM users WHERE id = a.target_id)
		ELSE 'Все сотрудники' END, ''),
	a.requirement, a.due_date, COALESCE(a.assigned_by, ''), a.created_at
	FROM document_assignments a`

func scanAssignment(row pgx.Row) (model.Assignment, error) {
	var a model.Assignment
	err := row.Scan(&a.ID, &a.DocumentID, &a.TargetType, &a.TargetID, &a.TargetName,
		&a.Requirement, &a.DueDate, &a.AssignedBy, &a.CreatedAt)
	return a, err
}

// ListAssignments filters by document and/or target (empty = any).
func (p *Postgres) ListAssignments(ctx context.Context, documentID, targetType, targetID string) ([]model.Assignment, error) {
	rows, err := p.pool.Query(ctx, assignmentSelect+`
		WHERE ($1 = '' OR a.document_id = $1)
		  AND ($2 = '' OR a.target_type = $2)
		  AND ($3 = '' OR a.target_id = $3)
		ORDER BY a.created_at DESC`, documentID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Assignment{}
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpsertAssignment creates or updates the assignment for (document, target).
// created reports whether it is new (to notify only on new assignments).
func (p *Postgres) UpsertAssignment(ctx context.Context, a model.Assignment) (model.Assignment, bool, error) {
	if a.TargetType == "all" {
		a.TargetID = ""
	}
	var created bool
	err := p.pool.QueryRow(ctx, `INSERT INTO document_assignments
		(id, document_id, target_type, target_id, requirement, due_date, assigned_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (document_id, target_type, target_id) DO UPDATE
		SET requirement = EXCLUDED.requirement, due_date = EXCLUDED.due_date
		RETURNING id, (xmax = 0)`,
		newID("asg"), a.DocumentID, a.TargetType, a.TargetID,
		defaultStr(a.Requirement, "required"), a.DueDate, nilIfEmpty(a.AssignedBy)).Scan(&a.ID, &created)
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23503" {
			return model.Assignment{}, false, ErrNotFound
		}
		return model.Assignment{}, false, err
	}
	out, err := scanAssignment(p.pool.QueryRow(ctx, assignmentSelect+` WHERE a.id=$1`, a.ID))
	return out, created, err
}

func (p *Postgres) DeleteAssignment(ctx context.Context, id string) error {
	res, err := p.pool.Exec(ctx, `DELETE FROM document_assignments WHERE id=$1`, id)
	if err == nil && res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// UsersForTarget returns active users covered by an assignment target.
func (p *Postgres) UsersForTarget(ctx context.Context, targetType, targetID string) ([]string, error) {
	q := `SELECT id FROM users WHERE active AND role = 'employee'`
	args := []any{}
	switch targetType {
	case "department":
		q = `SELECT id FROM users WHERE active AND department_id = $1`
		args = append(args, targetID)
	case "user":
		q = `SELECT id FROM users WHERE active AND id = $1`
		args = append(args, targetID)
	}
	rows, err := p.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// PlanFor resolves a user's onboarding plan over active documents: the
// strongest requirement wins (required > optional), personal assignments
// override department ones, which override "all". Unassigned active
// documents form the optional library.
func (p *Postgres) PlanFor(ctx context.Context, userID string) ([]model.PlanItem, error) {
	rows, err := p.pool.Query(ctx, `
		WITH me AS (SELECT department_id FROM users WHERE id = $1),
		hits AS (
			SELECT a.document_id, a.requirement, a.target_type, a.due_date, a.created_at,
			       CASE a.target_type WHEN 'user' THEN 3 WHEN 'department' THEN 2 ELSE 1 END AS prio
			FROM document_assignments a, me
			WHERE a.target_type = 'all'
			   OR (a.target_type = 'department' AND a.target_id = me.department_id)
			   OR (a.target_type = 'user' AND a.target_id = $1)
		),
		best AS (
			SELECT DISTINCT ON (document_id) document_id, requirement, target_type, due_date, created_at
			FROM hits
			ORDER BY document_id, (requirement = 'required') DESC, prio DESC, due_date NULLS LAST
		)
		SELECT d.id, COALESCE(b.requirement, 'library'), COALESCE(b.target_type, 'library'),
		       b.due_date, b.created_at
		FROM documents d LEFT JOIN best b ON b.document_id = d.id
		WHERE d.visibility = 'active'
		ORDER BY d.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PlanItem{}
	for rows.Next() {
		var it model.PlanItem
		if err := rows.Scan(&it.DocumentID, &it.Requirement, &it.Source, &it.DueDate, &it.AssignedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CountAssignments reports whether any assignment exists (for seeding).
func (p *Postgres) CountAssignments(ctx context.Context) (int, error) {
	var n int
	err := p.pool.QueryRow(ctx, `SELECT count(*) FROM document_assignments`).Scan(&n)
	return n, err
}

// ---------- notifications ----------

func (p *Postgres) CreateNotifications(ctx context.Context, userIDs []string, kind, documentID, title, body string) error {
	if len(userIDs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, uid := range userIDs {
		batch.Queue(`INSERT INTO notifications (id, user_id, kind, document_id, title, body)
			VALUES ($1,$2,$3,$4,$5,$6)`, newID("ntf"), uid, kind, nilIfEmpty(documentID), title, body)
	}
	return p.pool.SendBatch(ctx, batch).Close()
}

func (p *Postgres) ListNotifications(ctx context.Context, userID string, limit int) ([]model.Notification, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := p.pool.Query(ctx, `SELECT id, kind, COALESCE(document_id, ''), title, body, read_at, created_at
		FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Notification{}
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Kind, &n.DocumentID, &n.Title, &n.Body, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var unread int
	err = p.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id=$1 AND read_at IS NULL`, userID).Scan(&unread)
	return out, unread, err
}

// MarkNotificationsRead marks one (id != "") or all notifications as read.
func (p *Postgres) MarkNotificationsRead(ctx context.Context, userID, id string) error {
	_, err := p.pool.Exec(ctx, `UPDATE notifications SET read_at = now()
		WHERE user_id=$1 AND read_at IS NULL AND ($2 = '' OR id = $2)`, userID, id)
	return err
}

// DepartmentByName finds a department (for seeding).
func (p *Postgres) DepartmentByName(ctx context.Context, name string) (model.Department, error) {
	var d model.Department
	err := p.pool.QueryRow(ctx, `SELECT id, name, created_at FROM departments WHERE name=$1`, name).Scan(&d.ID, &d.Name, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}
