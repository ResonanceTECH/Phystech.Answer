package model

// Document is a regulation uploaded to the knowledge base.
type Document struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
	Number  string   `json:"number,omitempty"` // "794-1"
	Type    string   `json:"type,omitempty"`   // приказ|регламент|распоряжение|приложение|форма
	Date    string   `json:"date,omitempty"`   // YYYY-MM-DD
}

// Chunk is a searchable fragment of a document. Vector is filled by the
// embedder; chunks without a vector are only reachable via lexical search.
type Chunk struct {
	ID        string    `json:"id"`
	DocID     string    `json:"docId"`
	DocTitle  string    `json:"docTitle"`
	DocNumber string    `json:"docNumber,omitempty"`
	DocDate   string    `json:"docDate,omitempty"`
	DocType   string    `json:"docType,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Section   string    `json:"section,omitempty"` // citation label: "п. 3.1–3.4"
	Heading   string    `json:"heading,omitempty"` // section heading
	Context   string    `json:"context,omitempty"` // header used for retrieval
	Text      string    `json:"text"`
	Vector    []float32 `json:"vector,omitempty"`
}

// SearchText is what BM25 and the embedder see: context header + body.
func (c Chunk) SearchText() string {
	if c.Context == "" {
		return c.Text
	}
	return c.Context + "\n" + c.Text
}

// Source is a citation attached to an answer.
type Source struct {
	Ref        int     `json:"ref"` // [n] marker used in the answer text
	DocumentID string  `json:"documentId"`
	Title      string  `json:"title"`
	Number     string  `json:"number,omitempty"`
	Date       string  `json:"date,omitempty"`
	Section    string  `json:"section,omitempty"`
	Heading    string  `json:"heading,omitempty"`
	Excerpt    string  `json:"excerpt"`
	Score      float64 `json:"score"`
}

// Step is one action of a procedure with the sources that support it.
type Step struct {
	Text string `json:"text"`
	Refs []int  `json:"refs"`
}

// Conflict describes a contradiction or version change between documents.
type Conflict struct {
	Description string `json:"description"`
	Refs        []int  `json:"refs"`
}

// AskResponse is the result of POST /ask.
//
// Answer keeps a full markdown rendering for simple clients; structured
// fields (Status/Summary/Steps/Conflicts) are for the UI and evaluation.
type AskResponse struct {
	Status    string     `json:"status"` // answered|partial|not_found|invalid
	Summary   string     `json:"summary,omitempty"`
	Steps     []Step     `json:"steps,omitempty"`
	Notes     []string   `json:"notes,omitempty"`
	Conflicts []Conflict `json:"conflicts,omitempty"`
	Missing   string     `json:"missing,omitempty"`
	Answer    string     `json:"answer"`
	Sources   []Source   `json:"sources"`
	Trace     *Trace     `json:"trace,omitempty"`
}

// Trace exposes pipeline internals for debugging and the UI "как найден
// ответ" panel.
type Trace struct {
	Mode         string   `json:"mode"` // bm25|vector|hybrid
	Queries      []string `json:"queries,omitempty"`
	Candidates   int      `json:"candidates"`
	Reranked     bool     `json:"reranked"`
	Reranker     string   `json:"reranker,omitempty"` // llm | decision
	Amendments   []string `json:"amendments,omitempty"`
	Model        string   `json:"model,omitempty"`
	LatencyMs    int64    `json:"latencyMs"`
	RetrievalMs  int64    `json:"retrievalMs"`
	GenerationMs int64    `json:"generationMs"`
	DroppedRefs  int      `json:"droppedRefs,omitempty"`
}
