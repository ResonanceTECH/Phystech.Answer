package httpapi

import (
	"errors"
	"net/http"

	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

// POST /api/v1/answers/{id}/feedback — «полезно/неполезно» по спеке §16.
func (s *Server) answerFeedback(w http.ResponseWriter, r *http.Request) {
	u, _ := s.currentUser(r) // фидбэк принимаем и анонимно, но user_id кладём если есть
	var in struct {
		Rating  string `json:"rating"`
		Comment string `json:"comment"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Rating != "positive" && in.Rating != "negative" {
		writeError(w, http.StatusBadRequest, "rating must be positive or negative")
		return
	}
	if err := s.store.AddFeedback(r.Context(), r.PathValue("id"), u.ID, in.Rating, in.Comment); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save feedback")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// POST /api/v1/saved-instructions — сохранить проверенный ответ как инструкцию.
func (s *Server) createSavedInstruction(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		AnswerID string `json:"answerId"`
		Title    string `json:"title"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.AnswerID == "" {
		writeError(w, http.StatusBadRequest, "answerId is required")
		return
	}
	si, err := s.store.CreateSavedInstruction(r.Context(), model.SavedInstruction{
		UserID:   u.ID,
		AnswerID: in.AnswerID,
		Title:    in.Title,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save instruction")
		return
	}
	writeJSON(w, http.StatusCreated, si)
}

func (s *Server) listSavedInstructions(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	list, err := s.store.ListSavedInstructions(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list saved instructions")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// POST /api/v1/action-routes — интерактивный маршрут из ответа (§24).
// Шаги можно передать явно или скопировать из ответа.
func (s *Server) createActionRoute(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		AnswerID string `json:"answerId"`
		Title    string `json:"title"`
		Steps    []struct {
			StepID string `json:"stepId"`
			Order  int    `json:"order"`
			Title  string `json:"title"`
		} `json:"steps"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.AnswerID == "" {
		writeError(w, http.StatusBadRequest, "answerId is required")
		return
	}
	steps := make([]model.ActionRouteStep, 0, len(in.Steps))
	if len(in.Steps) == 0 {
		// Копируем шаги ответа — маршрут следует за проверенной инструкцией.
		answerSteps, err := s.store.ListAnswerSteps(r.Context(), in.AnswerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load answer steps")
			return
		}
		for _, st := range answerSteps {
			steps = append(steps, model.ActionRouteStep{
				StepID: st.ID,
				Order:  st.Order,
				Title:  st.Title,
			})
		}
	} else {
		for _, st := range in.Steps {
			steps = append(steps, model.ActionRouteStep{
				StepID: st.StepID,
				Order:  st.Order,
				Title:  st.Title,
			})
		}
	}
	route, err := s.store.CreateActionRoute(r.Context(), model.ActionRoute{
		UserID:   u.ID,
		AnswerID: in.AnswerID,
		Title:    in.Title,
	}, steps)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create route")
		return
	}
	writeJSON(w, http.StatusCreated, route)
}

func (s *Server) listActionRoutes(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	list, err := s.store.ListActionRoutes(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list routes")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getActionRoute(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	route, steps, err := s.store.GetActionRoute(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load route")
		return
	}
	if route.UserID != u.ID && u.Role != "admin" {
		writeError(w, http.StatusForbidden, "not your route")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"route": route, "steps": steps})
}

// PATCH /api/v1/action-routes/{id}/steps/{stepId} — отметить шаг выполненным.
func (s *Server) patchRouteStep(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Done bool `json:"done"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	route, _, err := s.store.GetActionRoute(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load route")
		return
	}
	if route.UserID != u.ID {
		writeError(w, http.StatusForbidden, "not your route")
		return
	}
	if err := s.store.SetRouteStepDone(r.Context(), route.ID, r.PathValue("stepId"), in.Done); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "step not found")
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update step")
	} else {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
