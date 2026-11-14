package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"phystech-answer/backend/internal/jobs"
	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/parse"
	"phystech-answer/backend/internal/store"
)

const maxUploadBytes = 64 << 20 // 64 MiB per request

type uploadResult struct {
	Filename     string `json:"filename"`
	Status       string `json:"status"` // queued|duplicate|rejected
	FileID       string `json:"fileId,omitempty"`
	DocumentID   string `json:"documentId,omitempty"`
	VersionID    string `json:"versionId,omitempty"`
	JobID        string `json:"jobId,omitempty"`
	Error        string `json:"error,omitempty"`
}

// uploadDocuments accepts multipart files (field "files" or "file"),
// stores them, creates document+version rows and enqueues parsing.
// Admin-only per spec §9.
func (s *Server) uploadDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	files := r.MultipartForm.File["files"]
	files = append(files, r.MultipartForm.File["file"]...)
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "no files in request")
		return
	}

	titleOverride := strings.TrimSpace(r.FormValue("title"))
	tags := splitCSV(r.FormValue("tags"))
	category := strings.TrimSpace(r.FormValue("category"))
	docType := strings.TrimSpace(r.FormValue("documentType"))
	docNumber := strings.TrimSpace(r.FormValue("documentNumber"))

	results := make([]uploadResult, 0, len(files))
	for _, fh := range files {
		results = append(results, s.processUpload(r, user, fh.Filename, func() (io.ReadCloser, error) {
			return fh.Open()
		}, titleOverride, tags, category, docType, docNumber))
	}

	status := http.StatusOK
	writeJSON(w, status, map[string]any{"results": results})
}

func (s *Server) processUpload(r *http.Request, user model.User, filename string, open func() (io.ReadCloser, error), titleOverride string, tags []string, category, docType, docNumber string) uploadResult {
	res := uploadResult{Filename: filename}
	ctx := r.Context()

	if !parse.Supported(parse.DetectByName(filename)) {
		res.Status = "rejected"
		res.Error = "unsupported file type"
		return res
	}

	rc, err := open()
	if err != nil {
		res.Status = "rejected"
		res.Error = "cannot read file"
		return res
	}
	defer rc.Close()

	h := sha256.New()
	data, err := io.ReadAll(io.TeeReader(io.LimitReader(rc, maxUploadBytes), h))
	if err != nil {
		res.Status = "rejected"
		res.Error = "cannot read file"
		return res
	}
	sum := hex.EncodeToString(h.Sum(nil))

	// Дубликат по контрольной сумме.
	if existing, err := s.store.GetSourceFileBySHA(ctx, sum); err == nil {
		res.Status = "duplicate"
		res.FileID = existing.ID
		return res
	} else if !errors.Is(err, store.ErrNotFound) {
		res.Status = "rejected"
		res.Error = "dedup check failed"
		return res
	}

	// Content-addressed key: identical files share one object.
	storageKey := "files/" + sum + filenameExt(filename)
	fileRec, err := s.store.CreateSourceFile(ctx, model.SourceFile{
		OriginalFilename: filename,
		StorageKey:       storageKey,
		MimeType:         mimeByExt(filename),
		SizeBytes:        int64(len(data)),
		SHA256:           sum,
		UploadedBy:       user.ID,
	})
	if err != nil {
		res.Status = "rejected"
		res.Error = "failed to register file"
		return res
	}
	res.FileID = fileRec.ID

	if err := s.fs.Save(ctx, storageKey, strings.NewReader(string(data))); err != nil {
		res.Status = "rejected"
		res.Error = "failed to store file"
		return res
	}

	title := titleOverride
	if title == "" {
		title = strings.TrimSuffix(filename, filenameExt(filename))
	}
	doc, err := s.store.CreateDocument(ctx, model.Document{
		Title:          title,
		DocumentNumber: docNumber,
		DocumentType:   docType,
		Category:       category,
		Tags:           tags,
		SourceFileID:   fileRec.ID,
	})
	if err != nil {
		res.Status = "rejected"
		res.Error = "failed to create document"
		return res
	}
	res.DocumentID = doc.ID

	vers, err := s.store.ListVersions(ctx, doc.ID)
	if err != nil || len(vers) == 0 {
		res.Status = "rejected"
		res.Error = "failed to create version"
		return res
	}
	version := vers[0]
	res.VersionID = version.ID

	if err := s.store.SetVersionStatus(ctx, version.ID, "pending", ""); err != nil {
		res.Status = "rejected"
		res.Error = "failed to set pending status"
		return res
	}

	payload, _ := json.Marshal(jobs.DocumentParsePayload{
		FileID: fileRec.ID, DocumentID: doc.ID, VersionID: version.ID,
		Filename: filename, StorageKey: storageKey,
	})
	job, err := s.store.EnqueueJob(ctx, model.Job{
		DocumentVersionID: version.ID,
		JobType:           jobs.JobDocumentParse,
		Payload:           payload,
	})
	if err != nil {
		res.Status = "rejected"
		res.Error = "failed to enqueue parse job"
		return res
	}
	res.JobID = job.ID
	res.Status = "queued"
	return res
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func filenameExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

func mimeByExt(name string) string {
	switch strings.ToLower(filenameExt(name)) {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// ---------- versions / structure / files ----------

func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	vers, err := s.store.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list versions")
		return
	}
	for i := range vers {
		vers[i].Content = "" // heavy content lives in /content
	}
	writeJSON(w, http.StatusOK, vers)
}

func (s *Server) listSections(w http.ResponseWriter, r *http.Request) {
	vers, err := s.store.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil || len(vers) == 0 {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	versionID := latestPublishedVersion(vers).ID
	secs, err := s.store.ListSections(r.Context(), versionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sections")
		return
	}
	writeJSON(w, http.StatusOK, secs)
}

func (s *Server) listChunks(w http.ResponseWriter, r *http.Request) {
	chunks, err := s.store.ListChunks(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chunks")
		return
	}
	writeJSON(w, http.StatusOK, chunks)
}

func (s *Server) listRelations(w http.ResponseWriter, r *http.Request) {
	rels, err := s.store.ListRelations(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list relations")
		return
	}
	writeJSON(w, http.StatusOK, rels)
}

func (s *Server) listAttachments(w http.ResponseWriter, r *http.Request) {
	rels, err := s.store.ListRelations(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list attachments")
		return
	}
	out := []model.DocumentRelation{}
	for _, rel := range rels {
		if rel.RelationType == "has_attachment" {
			out = append(out, rel)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func latestPublishedVersion(vers []model.DocumentVersion) model.DocumentVersion {
	for _, v := range vers {
		if v.ProcessingStatus == "published" {
			return v
		}
	}
	return vers[len(vers)-1]
}

// getDocumentContent returns the versioned content with structure — the
// machine-readable view used by citations.
func (s *Server) getDocumentContent(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.GetDocument(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load document")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// getDocumentFile streams the original uploaded file of the current version.
func (s *Server) getDocumentFile(w http.ResponseWriter, r *http.Request) {
	vers, err := s.store.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil || len(vers) == 0 {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	v := latestPublishedVersion(vers)
	if v.SourceFileID == "" {
		writeError(w, http.StatusNotFound, "no source file")
		return
	}
	s.serveSourceFile(w, r, v.SourceFileID)
}

func (s *Server) getSourceFile(w http.ResponseWriter, r *http.Request) {
	s.serveSourceFile(w, r, r.PathValue("id"))
}

func (s *Server) serveSourceFile(w http.ResponseWriter, r *http.Request, fileID string) {
	f, err := s.store.GetSourceFile(r.Context(), fileID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load file")
		return
	}
	rc, err := s.fs.Open(r.Context(), f.StorageKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "file missing in storage")
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", orDefault(f.MimeType, "application/octet-stream"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename*=UTF-8''%s", f.OriginalFilename))
	io.Copy(w, rc)
}
