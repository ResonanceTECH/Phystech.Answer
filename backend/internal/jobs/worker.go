// Package jobs implements the background worker loop from spec §19: jobs
// live in Postgres, are claimed with FOR UPDATE SKIP LOCKED, retried with
// backoff and recovered after a worker crash.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"phystech-answer/backend/internal/mlclient"
	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/parse"
	"phystech-answer/backend/internal/storage"
	"phystech-answer/backend/internal/store"
)

const (
	JobDocumentParse  = "document.parse"
	JobAnswerGenerate = "answer.generate"
)

// DocumentParsePayload is enqueued after a file upload.
type DocumentParsePayload struct {
	FileID     string `json:"file_id"`
	DocumentID string `json:"document_id"`
	VersionID  string `json:"version_id"`
	Filename   string `json:"filename"`
	StorageKey string `json:"storage_key"`
}

// AnswerGeneratePayload is enqueued when a conversation message needs an answer.
type AnswerGeneratePayload struct {
	RunID          string `json:"run_id"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	Question       string `json:"question"`
}

type Runner struct {
	st          store.Store
	fs          storage.Storage
	ml          *mlclient.Client
	interval    time.Duration
	concurrency int
	staleAfter  time.Duration
	retryDelay  time.Duration
}

func NewRunner(st store.Store, fs storage.Storage, ml *mlclient.Client, concurrency int) *Runner {
	if concurrency <= 0 {
		concurrency = 2
	}
	return &Runner{
		st: st, fs: fs, ml: ml,
		interval:    500 * time.Millisecond,
		concurrency: concurrency,
		staleAfter:  10 * time.Minute,
		retryDelay:  30 * time.Second,
	}
}

// Run polls for jobs until ctx is done. Intended as a goroutine.
func (r *Runner) Run(ctx context.Context) {
	if n, err := r.st.RequeueStaleJobs(ctx, r.staleAfter); err == nil && n > 0 {
		log.Printf("jobs: requeued %d stale jobs", n)
	}
	sem := make(chan struct{}, r.concurrency)
	var wg sync.WaitGroup
	tick := time.NewTicker(r.interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-tick.C:
		}
		batch, err := r.st.ClaimJobs(ctx, r.concurrency)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("jobs: claim: %v", err)
			}
			continue
		}
		for _, j := range batch {
			j := j
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				r.handle(ctx, j)
			}()
		}
	}
}

func (r *Runner) handle(ctx context.Context, j model.Job) {
	var err error
	switch j.JobType {
	case JobDocumentParse:
		err = r.handleDocumentParse(ctx, j)
	case JobAnswerGenerate:
		err = r.handleAnswerGenerate(ctx, j)
	default:
		log.Printf("jobs: unknown type %q", j.JobType)
		err = errors.New("unknown job type")
	}
	if err != nil {
		log.Printf("jobs: %s %s failed (attempt %d): %v", j.JobType, j.ID, j.Attempts, err)
		if ferr := r.st.FailJob(ctx, j.ID, errCode(err), r.retryDelay); ferr != nil {
			log.Printf("jobs: fail %s: %v", j.ID, ferr)
		}
		return
	}
	if err := r.st.CompleteJob(ctx, j.ID); err != nil {
		log.Printf("jobs: complete %s: %v", j.ID, err)
	}
}

func errCode(err error) string {
	code := strings.ToUpper(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(err.Error(), "_"))
	if len(code) > 60 {
		code = code[:60]
	}
	return strings.Trim(code, "_")
}

// ---------- document.parse ----------

func (r *Runner) handleDocumentParse(ctx context.Context, j model.Job) error {
	var p DocumentParsePayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return err
	}

	storageKey := p.StorageKey
	if storageKey == "" {
		if f, ferr := r.st.GetSourceFile(ctx, p.FileID); ferr == nil {
			storageKey = f.StorageKey
		}
	}
	rc, err := r.fs.Open(ctx, storageKey)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(rc, 64<<20))
	rc.Close()
	if err != nil {
		return err
	}

	ftype := parse.DetectByName(p.Filename)
	if !parse.Supported(ftype) {
		return r.st.SetVersionStatus(ctx, p.VersionID, "requires_review",
			"unsupported file type: "+p.Filename)
	}

	text, err := parse.Extract(ftype, data)
	switch {
	case errors.Is(err, parse.ErrNoText):
		return r.st.SetVersionStatus(ctx, p.VersionID, "requires_review",
			"no text extracted — file may need OCR")
	case err != nil:
		return err
	}

	if err := r.st.SetVersionContent(ctx, p.VersionID, text); err != nil {
		return err
	}
	if err := r.st.SetVersionStatus(ctx, p.VersionID, "published", ""); err != nil {
		return err
	}

	// Push the processed document to the ML index.
	if r.ml != nil {
		doc, derr := r.st.GetDocument(ctx, p.DocumentID)
		if derr == nil {
			if ierr := r.ml.IndexDocument(ctx, doc); ierr != nil {
				log.Printf("jobs: index %s in ML: %v", doc.ID, ierr)
			}
		}
	}
	r.detectRelations(ctx, p.VersionID, text)
	return nil
}

// ---------- нормативные связи (спека §11) ----------

// «Считать утратившим силу приказ № 1465-1», «отменяется приказ № 12»,
// «внести изменения в приказ № 555» — слабый эвристический детектор.
// Кандидаты уходят на проверку (review_status='pending'), эксперт
// подтверждает их через /api/v1/admin/document-relations/{id}/verify.
var relationPatterns = []struct {
	re     *regexp.Regexp
	relTyp string
}{
	{regexp.MustCompile(`(?i)(?:отмен\w+|утративш\w+\s+силу)[^.]{0,160}?приказ\w*\s*№?\s*(\d{3,6}(?:-\d+)?)`), "repeals"},
	{regexp.MustCompile(`(?i)внести\s+изменен\w+[^.]{0,160}?приказ\w*\s*№?\s*(\d{3,6}(?:-\d+)?)`), "amends"},
}

func (r *Runner) detectRelations(ctx context.Context, versionID, text string) {
	chunks, err := r.st.ListChunks(ctx, versionID)
	if err != nil {
		return
	}
	chunkBySection := map[string]string{}
	for _, c := range chunks {
		chunkBySection[strings.TrimRight(c.SectionNumber, ".")] = c.ID
	}
	for _, rp := range relationPatterns {
		for _, m := range rp.re.FindAllStringSubmatchIndex(text, -1) {
			num := text[m[2]:m[3]]
			target, err := r.st.FindDocumentByNumber(ctx, num)
			if err != nil {
				continue // документа нет в базе — спека требует не выдумывать
			}
			if _, err := r.st.CreateRelation(ctx, model.DocumentRelation{
				SourceVersionID:  versionID,
				TargetDocumentID: target.ID,
				RelationType:     rp.relTyp,
				ReviewStatus:     "pending",
			}); err != nil {
				log.Printf("jobs: create relation %s->%s: %v", versionID, target.ID, err)
			}
		}
	}
}

// ---------- answer.generate ----------

var stepLineRe = regexp.MustCompile(`^\s*(\d+)[.)]\s+(.+)$`)
var citeMarkRe = regexp.MustCompile(`\[(\d+)\]`)

func (r *Runner) handleAnswerGenerate(ctx context.Context, j model.Job) error {
	var p AnswerGeneratePayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return err
	}

	if cancelled, err := r.isCancelled(ctx, p.RunID); err != nil {
		return err
	} else if cancelled {
		return nil
	}
	if err := r.st.SetRunStatus(ctx, p.RunID, "running", ""); err != nil {
		return err
	}
	r.emit(ctx, p.RunID, "run.started", map[string]any{"run_id": p.RunID})
	r.emit(ctx, p.RunID, "retrieval.started", map[string]any{"run_id": p.RunID})

	res, err := r.askML(ctx, p.Question)
	if err != nil {
		r.emit(ctx, p.RunID, "answer.failed", map[string]any{"error": err.Error()})
		r.st.SetRunStatus(ctx, p.RunID, "failed", "ML_UNAVAILABLE")
		return err
	}
	r.emit(ctx, p.RunID, "retrieval.completed", map[string]any{"documents_found": len(res.Sources)})

	if cancelled, err := r.isCancelled(ctx, p.RunID); err != nil {
		return err
	} else if cancelled {
		return nil
	}
	r.emit(ctx, p.RunID, "verification.started", map[string]any{"run_id": p.RunID})

	answer, steps, citations, msg := r.buildAnswer(ctx, p, res)
	saved, err := r.st.CreateAnswer(ctx, answer, steps, citations)
	if err != nil {
		return err
	}
	if _, err := r.st.CreateMessage(ctx, model.Message{
		ConversationID: p.ConversationID, Role: "assistant", Content: msg,
	}); err != nil {
		return err
	}

	r.emit(ctx, p.RunID, "answer.completed", map[string]any{"answer_id": saved.ID})
	return r.st.SetRunStatus(ctx, p.RunID, "completed", "")
}

func (r *Runner) isCancelled(ctx context.Context, runID string) (bool, error) {
	run, err := r.st.GetRun(ctx, runID)
	if err != nil {
		return false, err
	}
	return run.Status == "cancelled", nil
}

func (r *Runner) emit(ctx context.Context, runID, typ string, payload any) {
	if err := r.st.AddRunEvent(ctx, runID, typ, payload); err != nil {
		log.Printf("jobs: emit %s for %s: %v", typ, runID, err)
	}
}

func (r *Runner) askML(ctx context.Context, question string) (mlclient.AskResult, error) {
	if r.ml == nil || !r.ml.Enabled() {
		return mlclient.AskResult{
			Answer: "ML-сервис пока недоступен, это ответ-заглушка.",
			Stub:   true,
		}, nil
	}
	res, err := r.ml.Ask(ctx, question)
	if err != nil {
		return mlclient.AskResult{
			Answer:  "ML-сервис пока недоступен, это ответ-заглушка.",
			Sources: []mlclient.Source{},
			Stub:    true,
		}, nil
	}
	return res, nil
}

// answerStatuses is the closed set of statuses persisted on answers.
var answerStatuses = map[string]bool{
	"answered": true, "partial": true, "clarification_required": true,
	"not_found": true, "conflict": true, "error": true,
}

// buildAnswer maps the ML response into the persistent answer model.
// When the ML returns a structured payload (status/steps/conflicts) it is
// used directly; otherwise numbered lines of the text become steps and
// [n] markers bind citations to steps.
func (r *Runner) buildAnswer(ctx context.Context, p AnswerGeneratePayload, res mlclient.AskResult) (model.Answer, []model.AnswerStep, []model.Citation, string) {
	status := res.Status
	if !answerStatuses[status] {
		status = "answered"
		if len(res.Sources) == 0 {
			status = "not_found"
		}
	}
	warnings := append([]string{}, res.Warnings...)
	if res.Stub {
		warnings = append(warnings, "Ответ сформирован в деградированном режиме без ML-сервиса.")
	}
	for _, c := range res.Conflicts {
		if strings.TrimSpace(c.Description) != "" {
			warnings = append(warnings, "Конфликт норм: "+c.Description)
		}
	}

	// Citations from sources; resolve version + chunk ids.
	citations := make([]model.Citation, 0, len(res.Sources))
	for _, src := range res.Sources {
		cit := model.Citation{
			DocumentID:    src.DocumentID,
			DocumentTitle: src.Title,
			SectionNumber: src.Section,
			Quote:         src.Excerpt,
			Score:         src.Score,
		}
		if src.Page > 0 {
			pg := src.Page
			cit.PageNumber = &pg
		}
		r.resolveCitationRefs(ctx, &cit)
		citations = append(citations, cit)
	}

	var steps []model.AnswerStep
	if len(res.Steps) > 0 {
		// Structured path: steps and per-step citation links come from ML.
		chunkToIdx := map[string]int{}
		for i, src := range res.Sources {
			if src.ChunkID != "" {
				chunkToIdx[src.ChunkID] = i
			}
		}
		for i, s := range res.Steps {
			order := s.Order
			if order <= 0 {
				order = i + 1
			}
			steps = append(steps, model.AnswerStep{
				Order: order, Title: s.Title, Description: s.Description,
				Responsible: s.Responsible, Deadline: s.Deadline,
				Condition: s.Condition, Required: true,
			})
			for _, cid := range s.CitationIDs {
				if idx, ok := chunkToIdx[cid]; ok {
					citations[idx].StepOrder = order
				}
			}
		}
	} else {
		// Legacy path: parse numbered lines of the answer text into steps.
		order := 0
		for _, line := range strings.Split(res.Answer, "\n") {
			m := stepLineRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			order++
			desc := strings.TrimSpace(m[2])
			steps = append(steps, model.AnswerStep{
				Order:       order,
				Title:       truncateRunes(desc, 80),
				Description: desc,
				Required:    true,
			})
			// Bind [n] markers found in this step's text to citation n.
			for _, cm := range citeMarkRe.FindAllStringSubmatch(m[2], -1) {
				if n, err := strconv.Atoi(cm[1]); err == nil && n >= 1 && n <= len(citations) {
					citations[n-1].StepOrder = order
				}
			}
		}
	}
	if len(steps) == 0 {
		steps = nil
	}

	missing := append([]string{}, res.MissingInformation...)
	if status == "not_found" && len(missing) == 0 {
		missing = append(missing, "В базе знаний не найдено подтверждающих фрагментов.")
	}

	answer := model.Answer{
		RunID:              p.RunID,
		Status:             status,
		Summary:            res.Answer,
		Warnings:           warnings,
		MissingInformation: missing,
	}
	return answer, steps, citations, res.Answer
}

// resolveCitationRefs links a citation to the document version and chunk
// that produced the source fragment (best effort — misses stay NULL).
func (r *Runner) resolveCitationRefs(ctx context.Context, cit *model.Citation) {
	if cit.DocumentID == "" {
		return
	}
	vers, err := r.st.ListVersions(ctx, cit.DocumentID)
	if err != nil || len(vers) == 0 {
		return
	}
	var versionID string
	for _, v := range vers {
		if v.ProcessingStatus == "published" {
			versionID = v.ID
		}
	}
	if versionID == "" {
		versionID = vers[len(vers)-1].ID
	}
	cit.DocumentVersionID = versionID

	want := strings.TrimRight(cit.SectionNumber, ".")
	if want == "" {
		return
	}
	chunks, err := r.st.ListChunks(ctx, versionID)
	if err != nil {
		return
	}
	for _, c := range chunks {
		if strings.TrimRight(c.SectionNumber, ".") == want {
			cit.ChunkID = c.ID
			return
		}
	}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
