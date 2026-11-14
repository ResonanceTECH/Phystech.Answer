package model

// Document is a regulation uploaded to the knowledge base.
type Document struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
}

// Chunk is a searchable fragment of a document. Vector is filled by the
// embedder; chunks without a vector are only reachable via lexical search.
type Chunk struct {
	ID       string    `json:"id"`
	DocID    string    `json:"docId"`
	DocTitle string    `json:"docTitle"`
	Tags     []string  `json:"tags,omitempty"`
	Section  string    `json:"section,omitempty"`
	Text     string    `json:"text"`
	Vector   []float32 `json:"vector,omitempty"`
}

// Source is a citation attached to an answer.
type Source struct {
	DocumentID string  `json:"documentId"`
	Title      string  `json:"title"`
	Section    string  `json:"section,omitempty"`
	Excerpt    string  `json:"excerpt"`
	Score      float64 `json:"score"`
}

// AskResponse is the result of POST /ask.
type AskResponse struct {
	Answer  string   `json:"answer"`
	Sources []Source `json:"sources"`
}
