package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

// Онбординг и управление базой знаний: папки и видимость документов,
// отделы, участники, назначения документов, план сотрудника, уведомления.

var validVisibility = map[string]bool{"active": true, "hidden": true, "archived": true}

// ---------- documents (admin) ----------

type patchDocumentRequest struct {
	Title      *string `json:"title"`
	Folder     *string `json:"folder"`
	Visibility *string `json:"visibility"`
}

// patchDocument: переименовать, перенести в папку, скрыть/вернуть/архивировать.
// Скрытый или архивный документ сразу убирается из индекса ассистента.
func (s *Server) patchDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var req patchDocumentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Visibility != nil && !validVisibility[*req.Visibility] {
		writeError(w, http.StatusBadRequest, "visibility must be active, hidden or archived")
		return
	}
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" {
			writeError(w, http.StatusBadRequest, "title must not be empty")
			return
		}
		req.Title = &t
	}
	id := r.PathValue("id")
	before, err := s.store.GetDocument(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load document")
		return
	}
	if err := s.store.UpdateDocumentAdmin(r.Context(), id, req.Title, req.Folder, req.Visibility); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update document")
		return
	}
	doc, _ := s.store.GetDocument(r.Context(), id)
	switch {
	case doc.Visibility != "active" && before.Visibility == "active":
		s.removeFromIndex(r.Context(), id)
	case doc.Visibility == "active" && (before.Visibility != "active" || req.Title != nil):
		s.indexDocument(r.Context(), doc)
	}
	_ = s.store.AddAuditLog(r.Context(), user.ID, "document", id, "update", "",
		map[string]string{"title": before.Title, "folder": before.Category, "visibility": before.Visibility},
		map[string]string{"title": doc.Title, "folder": doc.Category, "visibility": doc.Visibility})
	doc.Content = ""
	writeJSON(w, http.StatusOK, doc)
}

// ---------- departments ----------

type departmentRequest struct {
	Name string `json:"name"`
}

func (s *Server) createDepartment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req departmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	d, err := s.store.CreateDepartment(r.Context(), req.Name)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "department already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create department")
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) renameDepartment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req departmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	err := s.store.RenameDepartment(r.Context(), r.PathValue("id"), req.Name)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "department not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "department already exists")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to rename department")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	err := s.store.DeleteDepartment(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "department not found")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to delete department")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// ---------- participants (admin) ----------

// listParticipants returns users with department and onboarding progress.
func (s *Server) listParticipants(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	users, err := s.store.ListUserProfiles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	for i := range users {
		s.fillProgress(r.Context(), &users[i])
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) fillProgress(ctx context.Context, u *model.UserProfile) {
	plan, err := s.store.PlanFor(ctx, u.ID)
	if err != nil {
		return
	}
	reads, _ := s.store.ReadStatus(ctx, u.ID)
	for _, it := range plan {
		_, read := reads[it.DocumentID]
		switch it.Requirement {
		case "required":
			u.RequiredTotal++
			if read {
				u.RequiredRead++
			}
		case "optional":
			u.OptionalTotal++
			if read {
				u.OptionalRead++
			}
		}
	}
}

type participantRequest struct {
	Name         string  `json:"name"`
	Login        string  `json:"login"`
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	Role         *string `json:"role"`
	DepartmentID *string `json:"departmentId"`
	Position     *string `json:"position"`
	Active       *bool   `json:"active"`
}

var validRoles = map[string]bool{"employee": true, "admin": true, "expert": true}

func (s *Server) createParticipant(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req participantRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name, req.Login = strings.TrimSpace(req.Name), strings.TrimSpace(req.Login)
	if req.Name == "" || req.Login == "" || len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "name, login and password (6+ chars) are required")
		return
	}
	role := "employee"
	if req.Role != nil {
		if !validRoles[*req.Role] {
			writeError(w, http.StatusBadRequest, "invalid role")
			return
		}
		role = *req.Role
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	u, err := s.store.CreateUser(r.Context(), model.User{Name: req.Name, Login: req.Login, Email: req.Email, Role: role}, string(hash))
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "login already taken")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	if req.DepartmentID != nil || req.Position != nil {
		if err := s.store.UpdateUserOrg(r.Context(), u.ID, nil, req.DepartmentID, req.Position, nil); err != nil {
			writeError(w, http.StatusBadRequest, "user created, but department is invalid")
			return
		}
	}
	p, _ := s.store.GetUserProfile(r.Context(), u.ID)
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) patchParticipant(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var req participantRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	id := r.PathValue("id")
	if req.Role != nil && !validRoles[*req.Role] {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}
	// Защита от потери доступа: нельзя заблокировать себя или снять с себя роль.
	if id == admin.ID && ((req.Active != nil && !*req.Active) || (req.Role != nil && *req.Role == "employee")) {
		writeError(w, http.StatusBadRequest, "cannot deactivate or demote yourself")
		return
	}
	err := s.store.UpdateUserOrg(r.Context(), id, req.Role, req.DepartmentID, req.Position, req.Active)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user or department not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	p, _ := s.store.GetUserProfile(r.Context(), id)
	s.fillProgress(r.Context(), &p)
	writeJSON(w, http.StatusOK, p)
}

// ---------- assignments (admin) ----------

func (s *Server) listAssignments(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	q := r.URL.Query()
	list, err := s.store.ListAssignments(r.Context(), q.Get("documentId"), q.Get("targetType"), q.Get("targetId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assignments")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type assignRequest struct {
	DocumentIDs []string `json:"documentIds"`
	TargetType  string   `json:"targetType"` // all|department|user
	TargetID    string   `json:"targetId"`
	Requirement string   `json:"requirement"` // required|optional
	DueDate     string   `json:"dueDate"`     // YYYY-MM-DD, optional
}

// createAssignments назначает документы всем, отделу или сотруднику и
// уведомляет тех, кому назначение пришло впервые.
func (s *Server) createAssignments(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var req assignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.DocumentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "documentIds are required")
		return
	}
	switch req.TargetType {
	case "all":
	case "department", "user":
		if req.TargetID == "" {
			writeError(w, http.StatusBadRequest, "targetId is required")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "targetType must be all, department or user")
		return
	}
	if req.Requirement == "" {
		req.Requirement = "required"
	}
	if req.Requirement != "required" && req.Requirement != "optional" {
		writeError(w, http.StatusBadRequest, "requirement must be required or optional")
		return
	}
	var due *time.Time
	if req.DueDate != "" {
		t, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "dueDate must be YYYY-MM-DD")
			return
		}
		due = &t
	}

	out, err := s.assign(r.Context(), admin.ID, req.DocumentIDs, req.TargetType, req.TargetID, req.Requirement, due, true)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign documents")
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// assign upserts assignments and (optionally) notifies affected users.
func (s *Server) assign(ctx context.Context, by string, docIDs []string, targetType, targetID, requirement string, due *time.Time, notify bool) ([]model.Assignment, error) {
	var out []model.Assignment
	var users []string
	if notify {
		users, _ = s.store.UsersForTarget(ctx, targetType, targetID)
	}
	for _, docID := range docIDs {
		a, created, err := s.store.UpsertAssignment(ctx, model.Assignment{
			DocumentID: docID, TargetType: targetType, TargetID: targetID,
			Requirement: requirement, DueDate: due, AssignedBy: by,
		})
		if err != nil {
			return out, err
		}
		out = append(out, a)
		if !created || len(users) == 0 {
			continue
		}
		doc, err := s.store.GetDocument(ctx, docID)
		if err != nil || doc.Visibility != "active" {
			continue
		}
		title := "Вам рекомендован документ"
		body := fmt.Sprintf("«%s» — рекомендуется к ознакомлению.", doc.Title)
		if requirement == "required" {
			title = "Вам назначен документ для ознакомления"
			body = fmt.Sprintf("«%s» — обязательно к ознакомлению", doc.Title)
			if due != nil {
				body += " до " + due.Format("02.01.2006")
			}
			body += "."
		}
		if err := s.store.CreateNotifications(ctx, users, "assigned", docID, title, body); err != nil {
			log.Printf("notify assigned %s: %v", docID, err)
		}
	}
	return out, nil
}

func (s *Server) deleteAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	err := s.store.DeleteAssignment(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "assignment not found")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to delete assignment")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// ---------- employee: plan & notifications ----------

type planDocument struct {
	model.PlanItem
	Title          string     `json:"title"`
	DocumentNumber string     `json:"documentNumber,omitempty"`
	DocumentType   string     `json:"documentType,omitempty"`
	DocumentDate   *time.Time `json:"documentDate,omitempty"`
	Folder         string     `json:"folder,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	Read           bool       `json:"read"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

// myPlan: обязательные (по отделу, лично, всем), рекомендуемые и остальная
// библиотека активных документов — со статусом ознакомления.
func (s *Server) myPlan(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	plan, err := s.store.PlanFor(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build plan")
		return
	}
	docs, err := s.store.ListDocuments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	byID := make(map[string]model.Document, len(docs))
	for _, d := range docs {
		byID[d.ID] = d
	}
	reads, _ := s.store.ReadStatus(r.Context(), user.ID)
	profile, _ := s.store.GetUserProfile(r.Context(), user.ID)

	items := make([]planDocument, 0, len(plan))
	for _, it := range plan {
		d, ok := byID[it.DocumentID]
		if !ok {
			continue
		}
		pd := planDocument{PlanItem: it, Title: d.Title, DocumentNumber: d.DocumentNumber,
			DocumentType: d.DocumentType, DocumentDate: d.DocumentDate, Folder: d.Category, UpdatedAt: d.UpdatedAt}
		if t, ok := reads[d.ID]; ok {
			pd.Read, pd.ReadAt = true, &t
		}
		items = append(items, pd)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":       profile,
		"department": profile.DepartmentName,
		"items":      items,
	})
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, unread, err := s.store.ListNotifications(r.Context(), user.ID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "unread": unread})
}

type readNotificationsRequest struct {
	ID string `json:"id"` // empty — mark all
}

func (s *Server) readNotifications(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req readNotificationsRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	if err := s.store.MarkNotificationsRead(r.Context(), user.ID, req.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update notifications")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// WaitForProcessing blocks until no document version is pending parsing
// (or the timeout passes).
func (s *Server) WaitForProcessing(ctx context.Context, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		docs, err := s.store.ListDocuments(ctx)
		if err == nil {
			pending := 0
			for _, d := range docs {
				if d.Processing == "pending" || d.Processing == "processing" {
					pending++
				}
			}
			if pending == 0 && len(docs) > 0 {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}
