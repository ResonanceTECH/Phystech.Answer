package model

// Document is a regulation uploaded to the knowledge base.
// The optional regulatory fields (number/type/date/status) come from the
// backend document model; they drive edition-awareness (repealed docs are
// annotated, not silently trusted).
type Document struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	Tags           []string `json:"tags,omitempty"`
	DocumentNumber string   `json:"documentNumber,omitempty"`
	DocumentType   string   `json:"documentType,omitempty"`
	DocumentDate   string   `json:"documentDate,omitempty"`
	Category       string   `json:"category,omitempty"`
	LegalStatus    string   `json:"legalStatus,omitempty"`
}

// Chunk is a searchable fragment of a document. Vector is filled by the
// embedder; chunks without a vector are only reachable via lexical search.
type Chunk struct {
	ID       string    `json:"id"`
	DocID    string    `json:"docId"`
	DocTitle string    `json:"docTitle"`
	Tags     []string  `json:"tags,omitempty"`
	Section  string    `json:"section,omitempty"`
	Page     int       `json:"page,omitempty"`
	Text     string    `json:"text"`
	Vector   []float32 `json:"vector,omitempty"`
}

// Source is a citation attached to an answer.
type Source struct {
	DocumentID     string  `json:"documentId"`
	ChunkID        string  `json:"chunkId,omitempty"`
	Title          string  `json:"title"`
	Section        string  `json:"section,omitempty"`
	Page           int     `json:"page,omitempty"`
	DocumentNumber string  `json:"documentNumber,omitempty"`
	LegalStatus    string  `json:"legalStatus,omitempty"`
	Excerpt        string  `json:"excerpt"`
	Score          float64 `json:"score"`
	RerankScore    float64 `json:"rerankScore,omitempty"`
}

// AnswerStep is one ordered action inside a structured answer.
type AnswerStep struct {
	Order       int      `json:"order"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Responsible *string  `json:"responsible,omitempty"`
	Deadline    *string  `json:"deadline,omitempty"`
	Condition   *string  `json:"condition,omitempty"`
	CitationIDs []string `json:"citation_ids"`
}

// Conflict describes contradictory normative provisions found in context.
type Conflict struct {
	Description string   `json:"description"`
	ChunkIDs    []string `json:"chunk_ids"`
}

// AnswerPayload is the structured ML output consumed by the Go backend
// (spec §13). /api/ask additionally mirrors it into the legacy text fields.
type AnswerPayload struct {
	Status             string       `json:"status"` // answered|partial|clarification_required|not_found|conflict|error
	Summary            string       `json:"summary"`
	Steps              []AnswerStep `json:"steps"`
	Warnings           []string     `json:"warnings"`
	MissingInformation []string     `json:"missing_information"`
	Conflicts          []Conflict   `json:"conflicts"`
	Sources            []Source     `json:"sources"`
	// Debug keeps intermediate candidates for offline evaluation.
	Debug *AnswerDebug `json:"debug,omitempty"`
}

// AnswerDebug records retrieval/rerank traces for evaluation runs.
type AnswerDebug struct {
	Retrieved []Source `json:"retrieved,omitempty"`
	Reranked  []Source `json:"reranked,omitempty"`
	Model     string   `json:"model,omitempty"`
	LatencyMS int64    `json:"latency_ms,omitempty"`
}

// Relation describes a normative link between documents
// (repeals/amends/approves/references/has_attachment).
type Relation struct {
	RelationType         string `json:"relation_type"`
	TargetDocumentID     string `json:"target_document_id,omitempty"`
	TargetDocumentNumber string `json:"target_document_number,omitempty"`
	TargetDocumentDate   string `json:"target_document_date,omitempty"`
	EvidenceChunkID      string `json:"evidence_chunk_id,omitempty"`
	Evidence             string `json:"evidence,omitempty"`
	Confirmed            bool   `json:"confirmed"`
}

// AskResponse is the result of POST /api/ask (legacy contract + extensions).
type AskResponse struct {
	Answer             string       `json:"answer"`
	Sources            []Source     `json:"sources"`
	Status             string       `json:"status,omitempty"`
	Steps              []AnswerStep `json:"steps,omitempty"`
	Warnings           []string     `json:"warnings,omitempty"`
	MissingInformation []string     `json:"missing_information,omitempty"`
	Conflicts          []Conflict   `json:"conflicts,omitempty"`
}
