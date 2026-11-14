package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"phystech-answer/backend/internal/jobs"
	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

// ---------- processing jobs ----------

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.store.ListJobs(r.Context(), r.URL.Query().Get("status"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	list, err := s.store.ListJobs(r.Context(), "", 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}
	for _, j := range list {
		if j.ID == r.PathValue("id") {
			writeJSON(w, http.StatusOK, j)
			return
		}
	}
	writeError(w, http.StatusNotFound, "job not found")
}

func (s *Server) retryJob(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	err := s.store.RetryJob(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found or not in a retryable state")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retry job")
		return
	}
	_ = s.store.AddAuditLog(r.Context(), user.ID, "processing_jobs", r.PathValue("id"), "retry", "", nil, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "queued"})
}

// ---------- publish / reprocess ----------

func (s *Server) publishDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	vers, err := s.store.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil || len(vers) == 0 {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	v := latestPublishedVersion(vers)
	if err := s.store.SetVersionStatus(r.Context(), v.ID, "published", ""); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to publish")
		return
	}
	doc, err := s.store.GetDocument(r.Context(), r.PathValue("id"))
	if err == nil && s.ml != nil {
		_ = s.ml.IndexDocument(r.Context(), doc)
	}
	_ = s.store.AddAuditLog(r.Context(), user.ID, "documents", doc.ID, "publish", "", nil, nil)
	writeJSON(w, http.StatusOK, doc)
}

// reprocessDocument re-enqueues parsing from the stored source file.
func (s *Server) reprocessDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	vers, err := s.store.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil || len(vers) == 0 {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	v := vers[len(vers)-1]
	if v.SourceFileID == "" {
		writeError(w, http.StatusBadRequest, "document has no source file to reprocess")
		return
	}
	f, err := s.store.GetSourceFile(r.Context(), v.SourceFileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source file missing")
		return
	}
	if err := s.store.SetVersionStatus(r.Context(), v.ID, "pending", ""); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset status")
		return
	}
	payload, _ := json.Marshal(jobs.DocumentParsePayload{
		FileID: f.ID, DocumentID: v.DocumentID, VersionID: v.ID,
		Filename: f.OriginalFilename, StorageKey: f.StorageKey,
	})
	job, err := s.store.EnqueueJob(r.Context(), model.Job{
		DocumentVersionID: v.ID, JobType: jobs.JobDocumentParse, Payload: payload,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}
	_ = s.store.AddAuditLog(r.Context(), user.ID, "documents", v.DocumentID, "reprocess", "", nil, nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": job.ID, "status": "queued"})
}

// ---------- document relations ----------

type relationRequest struct {
	SourceVersionID  string `json:"sourceVersionId"`
	TargetDocumentID string `json:"targetDocumentId"`
	TargetVersionID  string `json:"targetVersionId"`
	RelationType     string `json:"relationType"`
	ReviewStatus     string `json:"reviewStatus"`
}

var validRelationTypes = map[string]bool{
	"approves": true, "amends": true, "repeals": true, "replaces": true,
	"supplements": true, "references": true, "has_attachment": true,
}

func (s *Server) createRelation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req relationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validRelationTypes[req.RelationType] {
		writeError(w, http.StatusBadRequest, "invalid relationType")
		return
	}
	rel, err := s.store.CreateRelation(r.Context(), model.DocumentRelation{
		SourceVersionID:  req.SourceVersionID,
		TargetDocumentID: req.TargetDocumentID,
		TargetVersionID:  req.TargetVersionID,
		RelationType:     req.RelationType,
		ReviewStatus:     "confirmed",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create relation")
		return
	}
	writeJSON(w, http.StatusCreated, rel)
}

func (s *Server) patchRelation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req struct {
		ReviewStatus string `json:"reviewStatus"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	s.verifyRelationWithStatus(w, r, req.ReviewStatus)
}

func (s *Server) verifyRelation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	s.verifyRelationWithStatus(w, r, "confirmed")
}

func (s *Server) verifyRelationWithStatus(w http.ResponseWriter, r *http.Request, status string) {
	if status == "" {
		status = "confirmed"
	}
	if !validReviewStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid reviewStatus")
		return
	}
	err := s.store.VerifyRelation(r.Context(), r.PathValue("id"), status)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "relation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify relation")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

func validReviewStatus(s string) bool {
	return s == "pending" || s == "confirmed" || s == "rejected"
}

// ---------- dashboard ----------

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	docs, err := s.store.ListDocuments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load documents")
		return
	}
	jobsAll, err := s.store.ListJobs(r.Context(), "", 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load jobs")
		return
	}
	jobStats := map[string]int{}
	for _, j := range jobsAll {
		jobStats[j.Status]++
	}
	procStats := map[string]int{}
	for _, d := range docs {
		procStats[orDefault(d.Processing, "unknown")]++
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documents":         len(docs),
		"processing_status": procStats,
		"jobs":              jobStats,
	})
}
