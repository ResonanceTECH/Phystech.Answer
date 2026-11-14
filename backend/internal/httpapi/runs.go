package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

// requireRunAccess verifies the caller owns the conversation that produced
// the run (admins/experts may inspect any run).
func (s *Server) requireRunAccess(w http.ResponseWriter, r *http.Request, runID string) (model.AnswerRun, bool) {
	run, err := s.store.GetRun(r.Context(), runID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "run not found")
		return run, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return run, false
	}
	user, ok := s.requireUser(w, r)
	if !ok {
		return run, false
	}
	if run.ConversationID != "" {
		conv, err := s.store.GetConversation(r.Context(), run.ConversationID)
		if err == nil && conv.UserID != user.ID && !isPrivileged(user) {
			writeError(w, http.StatusNotFound, "run not found")
			return run, false
		}
	}
	return run, true
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	run, ok := s.requireRunAccess(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) cancelRun(w http.ResponseWriter, r *http.Request) {
	run, ok := s.requireRunAccess(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	if run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled" {
		writeJSON(w, http.StatusOK, run)
		return
	}
	if err := s.store.SetRunStatus(r.Context(), run.ID, "cancelled", ""); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel run")
		return
	}
	_ = s.store.AddRunEvent(r.Context(), run.ID, "run.cancelled", map[string]any{"run_id": run.ID})
	run.Status = "cancelled"
	writeJSON(w, http.StatusOK, run)
}

// runEvents streams run events over SSE (spec §18). Events are persisted
// in run_events, so a client can reconnect and resume via Last-Event-ID.
func (s *Server) runEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if _, ok := s.requireRunAccess(w, r, runID); !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	lastID := int64(0)
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			lastID = n
		}
	}
	if v := r.URL.Query().Get("after"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > lastID {
			lastID = n
		}
	}

	ctx := r.Context()
	poll := time.NewTicker(400 * time.Millisecond)
	defer poll.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	terminal := func() bool {
		run, err := s.store.GetRun(ctx, runID)
		if err != nil {
			return true
		}
		return run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled"
	}

	for {
		events, err := s.store.ListRunEvents(ctx, runID, lastID)
		if err != nil {
			return
		}
		for _, e := range events {
			lastID = e.ID
			writeSSE(w, e)
		}
		if len(events) > 0 {
			flusher.Flush()
		}
		if terminal() {
			// Terminal state reached: drain remaining events once, then close.
			if rest, err := s.store.ListRunEvents(ctx, runID, lastID); err == nil {
				for _, e := range rest {
					writeSSE(w, e)
				}
			}
			flusher.Flush()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-poll.C:
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, e model.RunEvent) {
	fmt.Fprintf(w, "id: %d\n", e.ID)
	fmt.Fprintf(w, "event: %s\n", e.EventType)
	fmt.Fprintf(w, "data: %s\n\n", e.Payload)
}

// getRunAnswer returns the verified answer produced by a run.
func (s *Server) getRunAnswer(w http.ResponseWriter, r *http.Request) {
	run, ok := s.requireRunAccess(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	answer, steps, citations, err := s.store.GetAnswerByRun(r.Context(), run.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "answer not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get answer")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"answer_id":           answer.ID,
		"conversation_id":     answer.ConversationID,
		"status":              answer.Status,
		"summary":             answer.Summary,
		"steps":               steps,
		"sources":             citations,
		"warnings":            answer.Warnings,
		"missing_information": answer.MissingInformation,
	})
}
