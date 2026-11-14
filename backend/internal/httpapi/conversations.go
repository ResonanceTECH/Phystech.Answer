package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"phystech-answer/backend/internal/jobs"
	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

type createConversationRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// createConversation starts a dialog. If message is present it is saved and
// an answer run is queued — the response then includes run identifiers.
func (s *Server) createConversation(w http.ResponseWriter, r *http.Request) {
	var req createConversationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, _ := s.currentUser(r)

	conv, err := s.store.CreateConversation(r.Context(), model.Conversation{
		UserID: user.ID,
		Title:  orDefault(strings.TrimSpace(req.Title), "Новый диалог"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}

	if msg := strings.TrimSpace(req.Message); msg != "" {
		out, err := s.queueAnswerRun(r, conv.ID, user.ID, msg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to queue answer")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"conversation": conv,
			"message_id":   out.messageID,
			"run_id":       out.runID,
			"status":       "queued",
		})
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	convs, err := s.store.ListConversations(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	writeJSON(w, http.StatusOK, convs)
}

func (s *Server) getConversation(w http.ResponseWriter, r *http.Request) {
	conv, err := s.store.GetConversation(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	msgs, err := s.store.ListMessages(r.Context(), conv.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversation": conv,
		"messages":     msgs,
	})
}

type renameRequest struct {
	Title string `json:"title"`
}

func (s *Server) renameConversation(w http.ResponseWriter, r *http.Request) {
	var req renameRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	conv, err := s.store.UpdateConversationTitle(r.Context(), r.PathValue("id"), req.Title)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to rename conversation")
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (s *Server) deleteConversation(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteConversation(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete conversation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.store.ListMessages(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

type postMessageRequest struct {
	Content string `json:"content"`
}

type queuedRun struct {
	messageID string
	runID     string
}

// postMessage appends a user message and queues the answer run (spec §18).
func (s *Server) postMessage(w http.ResponseWriter, r *http.Request) {
	var req postMessageRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	conv, err := s.store.GetConversation(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	user, _ := s.currentUser(r)

	out, err := s.queueAnswerRun(r, conv.ID, user.ID, content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to queue answer")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"message_id": out.messageID,
		"run_id":     out.runID,
		"status":     "queued",
	})
}

// queueAnswerRun persists the user message, creates the run and enqueues
// the answer.generate job.
func (s *Server) queueAnswerRun(r *http.Request, convID, userID, content string) (queuedRun, error) {
	ctx := r.Context()
	msg, err := s.store.CreateMessage(ctx, model.Message{
		ConversationID: convID, Role: "user", Content: content,
	})
	if err != nil {
		return queuedRun{}, err
	}
	run, err := s.store.CreateRun(ctx, model.AnswerRun{
		ConversationID: convID, MessageID: msg.ID,
	})
	if err != nil {
		return queuedRun{}, err
	}
	payload, _ := json.Marshal(jobs.AnswerGeneratePayload{
		RunID: run.ID, ConversationID: convID, MessageID: msg.ID, Question: content,
	})
	if _, err := s.store.EnqueueJob(ctx, model.Job{
		JobType:        jobs.JobAnswerGenerate,
		Payload:        payload,
		IdempotencyKey: "answer:" + run.ID,
	}); err != nil {
		return queuedRun{}, err
	}
	return queuedRun{messageID: msg.ID, runID: run.ID}, nil
}
