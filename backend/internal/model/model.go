package model

import (
	"encoding/json"
	"time"
)

// Document is the API-facing regulation card. Content/Version are projected
// from the current published DocumentVersion to keep the frontend contract.
type Document struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	Tags           []string   `json:"tags"`
	Version        int        `json:"version"`
	DocumentNumber string     `json:"documentNumber,omitempty"`
	DocumentType   string     `json:"documentType,omitempty"`
	DocumentDate   *time.Time `json:"documentDate,omitempty"`
	Category       string     `json:"category,omitempty"`
	AccessLevel    string     `json:"accessLevel,omitempty"`
	LegalStatus    string     `json:"legalStatus,omitempty"`
	Processing     string     `json:"processingStatus,omitempty"`
	// Visibility: active | hidden | archived. Hidden and archived documents
	// are excluded from the assistant and from employees' lists.
	Visibility string    `json:"visibility,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	// SourceFileID links the initial version to an uploaded file (input-only).
	SourceFileID string `json:"-"`
}

type User struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	FirstName          string     `json:"firstName,omitempty"`
	LastName           string     `json:"lastName,omitempty"`
	Email              string     `json:"email"`
	Login              string     `json:"login,omitempty"`
	Role               string     `json:"role,omitempty"` // employee|admin|expert
	EmployeeCategory   string     `json:"employeeCategory,omitempty"`
	DepartmentID       string     `json:"departmentId,omitempty"`
	Position           string     `json:"position,omitempty"`
	OnboardingComplete bool       `json:"onboardingComplete"`
	EmailVerifiedAt    *time.Time `json:"emailVerifiedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
}

// ProfilePatch — поля, которые пользователь может менять сам (onboarding/profile).
// Системные права (role) сюда не входят и клиентом не назначаются.
type ProfilePatch struct {
	FirstName          *string `json:"firstName"`
	LastName           *string `json:"lastName"`
	EmployeeCategory   *string `json:"employeeCategory"`
	DepartmentID       *string `json:"departmentId"`
	Position           *string `json:"position"`
	OnboardingComplete *bool   `json:"onboardingComplete"`
}

type SourceFile struct {
	ID               string    `json:"id"`
	OriginalFilename string    `json:"originalFilename"`
	StorageKey       string    `json:"-"`
	MimeType         string    `json:"mimeType"`
	SizeBytes        int64     `json:"sizeBytes"`
	SHA256           string    `json:"sha256"`
	UploadedBy       string    `json:"uploadedBy,omitempty"`
	UploadedAt       time.Time `json:"uploadedAt"`
}

type DocumentVersion struct {
	ID               string     `json:"id"`
	DocumentID       string     `json:"documentId"`
	SourceFileID     string     `json:"sourceFileId,omitempty"`
	VersionNumber    int        `json:"versionNumber"`
	Content          string     `json:"content,omitempty"`
	EffectiveFrom    *time.Time `json:"effectiveFrom,omitempty"`
	EffectiveTo      *time.Time `json:"effectiveTo,omitempty"`
	LegalStatus      string     `json:"legalStatus"`
	ReviewStatus     string     `json:"reviewStatus"`
	ProcessingStatus string     `json:"processingStatus"`
	ProcessingError  string     `json:"processingError,omitempty"`
	PublishedAt      *time.Time `json:"publishedAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type Section struct {
	ID            string `json:"id"`
	VersionID     string `json:"versionId"`
	ParentID      string `json:"parentId,omitempty"`
	SectionNumber string `json:"sectionNumber"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	Position      int    `json:"position"`
}

type Chunk struct {
	ID            string `json:"id"`
	VersionID     string `json:"versionId"`
	SectionID     string `json:"sectionId,omitempty"`
	Content       string `json:"content"`
	ChunkIndex    int    `json:"chunkIndex"`
	LocationType  string `json:"locationType"`
	PageNumber    *int   `json:"pageNumber,omitempty"`
	SheetName     string `json:"sheetName,omitempty"`
	CellRange     string `json:"cellRange,omitempty"`
	SectionNumber string `json:"sectionNumber,omitempty"`
}

type DocumentRelation struct {
	ID               string     `json:"id"`
	SourceVersionID  string     `json:"sourceVersionId"`
	TargetDocumentID string     `json:"targetDocumentId"`
	TargetVersionID  string     `json:"targetVersionId,omitempty"`
	RelationType     string     `json:"relationType"`
	EvidenceChunkID  string     `json:"evidenceChunkId,omitempty"`
	EffectiveFrom    *time.Time `json:"effectiveFrom,omitempty"`
	ReviewStatus     string     `json:"reviewStatus"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId,omitempty"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	Role           string    `json:"role"` // user|assistant|system
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}

type AnswerRun struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	MessageID      string     `json:"messageId,omitempty"`
	Status         string     `json:"status"` // queued|running|completed|failed|cancelled
	ErrorCode      string     `json:"errorCode,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type RunEvent struct {
	ID        int64           `json:"id"`
	RunID     string          `json:"runId"`
	EventType string          `json:"event"`
	Payload   json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"createdAt"`
}

type Answer struct {
	ID                 string    `json:"id"`
	RunID              string    `json:"runId"`
	ConversationID     string    `json:"conversationId,omitempty"`
	Status             string    `json:"status"` // answered|partial|clarification_required|not_found|conflict|error
	Summary            string    `json:"summary"`
	Warnings           []string  `json:"warnings"`
	MissingInformation []string  `json:"missing_information"`
	CreatedAt          time.Time `json:"createdAt"`
}

type AnswerStep struct {
	ID          string   `json:"id"`
	AnswerID    string   `json:"-"`
	Order       int      `json:"order"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Responsible *string  `json:"responsible"`
	Deadline    *string  `json:"deadline"`
	Condition   *string  `json:"condition"`
	Required    bool     `json:"required"`
	CitationIDs []string `json:"citation_ids"`
}

type Citation struct {
	ID                string  `json:"id"`
	AnswerID          string  `json:"-"`
	StepID            string  `json:"step_id,omitempty"`
	ChunkID           string  `json:"chunk_id,omitempty"`
	DocumentVersionID string  `json:"document_version_id,omitempty"`
	DocumentID        string  `json:"document_id,omitempty"`
	DocumentTitle     string  `json:"document_title"`
	SectionNumber     string  `json:"section_number,omitempty"`
	PageNumber        *int    `json:"page_number,omitempty"`
	Quote             string  `json:"quote"`
	Score             float64 `json:"score"`
	// StepOrder is an input-only field: on CreateAnswer it binds the
	// citation to the step with this Order (0 = unbound).
	StepOrder int `json:"-"`
}

// SearchHit is one chunk-level result of hybrid/fulltext search.
type SearchHit struct {
	ChunkID       string  `json:"chunkId"`
	VersionID     string  `json:"versionId"`
	DocumentID    string  `json:"documentId"`
	DocumentTitle string  `json:"documentTitle"`
	SectionNumber string  `json:"sectionNumber,omitempty"`
	Content       string  `json:"content"`
	LegalStatus   string  `json:"legalStatus,omitempty"`
	AccessLevel   string  `json:"accessLevel,omitempty"`
	Rank          float64 `json:"rank"`
}

// SavedInstruction is a bookmarked verified answer.
type SavedInstruction struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	AnswerID  string    `json:"answerId"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
}

// ActionRoute is a user's personal checklist built from an answer (§24).
type ActionRoute struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	AnswerID  string    `json:"answerId"`
	Title     string    `json:"title"`
	Stale     bool      `json:"stale"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ActionRouteStep struct {
	ID      string     `json:"id"`
	RouteID string     `json:"routeId"`
	StepID  string     `json:"stepId,omitempty"`
	Order   int        `json:"order"`
	Title   string     `json:"title"`
	Done    bool       `json:"done"`
	DoneAt  *time.Time `json:"doneAt,omitempty"`
}

type Job struct {
	ID                string          `json:"id"`
	DocumentVersionID string          `json:"documentVersionId,omitempty"`
	JobType           string          `json:"jobType"`
	Status            string          `json:"status"`
	Payload           json.RawMessage `json:"payload,omitempty"`
	Progress          int             `json:"progress"`
	Attempts          int             `json:"attempts"`
	MaxAttempts       int             `json:"maxAttempts"`
	RetryAt           *time.Time      `json:"retryAt,omitempty"`
	ErrorCode         string          `json:"errorCode,omitempty"`
	IdempotencyKey    string          `json:"idempotencyKey,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	CompletedAt       *time.Time      `json:"completedAt,omitempty"`
}

// ---------- onboarding: departments, assignments, notifications ----------

type Department struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Members   int       `json:"members"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserProfile is a user with organisational data (admin views).
type UserProfile struct {
	User
	DepartmentID   string `json:"departmentId,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	Position       string `json:"position,omitempty"`
	Active         bool   `json:"active"`
	// Progress over required documents (filled by the API).
	RequiredTotal int `json:"requiredTotal"`
	RequiredRead  int `json:"requiredRead"`
	OptionalTotal int `json:"optionalTotal"`
	OptionalRead  int `json:"optionalRead"`
}

// Assignment targets: all employees, a department or one user.
type Assignment struct {
	ID          string     `json:"id"`
	DocumentID  string     `json:"documentId"`
	TargetType  string     `json:"targetType"` // all|department|user
	TargetID    string     `json:"targetId,omitempty"`
	TargetName  string     `json:"targetName,omitempty"`
	Requirement string     `json:"requirement"` // required|optional
	DueDate     *time.Time `json:"dueDate,omitempty"`
	AssignedBy  string     `json:"assignedBy,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// PlanItem is one document in an employee's onboarding plan.
type PlanItem struct {
	DocumentID  string     `json:"documentId"`
	Requirement string     `json:"requirement"` // required|optional|library
	Source      string     `json:"source"`      // all|department|user|library
	DueDate     *time.Time `json:"dueDate,omitempty"`
	AssignedAt  *time.Time `json:"assignedAt,omitempty"`
}

type Notification struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"` // assigned|new_document|updated
	DocumentID string     `json:"documentId,omitempty"`
	Title      string     `json:"title"`
	Body       string     `json:"body,omitempty"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}
