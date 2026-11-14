package store

import (
	"context"
	"errors"
	"time"

	"phystech-answer/backend/internal/model"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrTokenExpired = errors.New("token expired")
var ErrTokenUsed = errors.New("token already used")

// Store is the persistence boundary. The Postgres implementation lives in
// postgres.go; Memory is kept for tests and local dev without a database.
type Store interface {
	// Documents (API projection: doc + current published version content).
	ListDocuments(ctx context.Context) ([]model.Document, error)
	GetDocument(ctx context.Context, id string) (model.Document, error)
	CreateDocument(ctx context.Context, doc model.Document) (model.Document, error)
	UpdateDocument(ctx context.Context, doc model.Document) (model.Document, error)
	DeleteDocument(ctx context.Context, id string) error

	// Versions / structure.
	ListVersions(ctx context.Context, docID string) ([]model.DocumentVersion, error)
	GetVersion(ctx context.Context, versionID string) (model.DocumentVersion, error)
	ListSections(ctx context.Context, versionID string) ([]model.Section, error)
	ListChunks(ctx context.Context, versionID string) ([]model.Chunk, error)
	// SetVersionContent replaces version content and rebuilds sections+chunks.
	SetVersionContent(ctx context.Context, versionID, content string) error
	UpdateDocumentMeta(ctx context.Context, id, number, docType string, date *time.Time) error
	SetDocumentFolder(ctx context.Context, id, folder string) error
	SetVersionStatus(ctx context.Context, versionID, processingStatus, processingError string) error

	// Source files.
	CreateSourceFile(ctx context.Context, f model.SourceFile) (model.SourceFile, error)
	GetSourceFileBySHA(ctx context.Context, sha string) (model.SourceFile, error)
	GetSourceFile(ctx context.Context, id string) (model.SourceFile, error)
	// AttachVersionFile links a new uploaded file to a document as a new version.
	CreateVersion(ctx context.Context, v model.DocumentVersion) (model.DocumentVersion, error)

	// Users / auth.
	ListUsers(ctx context.Context) ([]model.User, error)
	CreateUser(ctx context.Context, user model.User, passwordHash string) (model.User, error)
	GetUserByLogin(ctx context.Context, login string) (model.User, string, error) // user, passwordHash
	GetUserByEmail(ctx context.Context, email string) (model.User, string, error) // user, passwordHash
	GetUser(ctx context.Context, id string) (model.User, error)
	UpdateUserProfile(ctx context.Context, id string, p model.ProfilePatch) (model.User, error)
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	CreateSession(ctx context.Context, userID, tokenHash string, ttl time.Duration) error
	GetSessionUser(ctx context.Context, tokenHash string) (model.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	RevokeUserSessions(ctx context.Context, userID string) error
	CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, ttl time.Duration) error
	// ConsumePasswordResetToken atomically marks the token used and returns
	// its owner. Errors: ErrNotFound | ErrTokenExpired | ErrTokenUsed.
	ConsumePasswordResetToken(ctx context.Context, tokenHash string) (model.User, error)
	ListDepartments(ctx context.Context) ([]model.Department, error)

	// Read marks.
	MarkRead(ctx context.Context, docID, userID string) (time.Time, error)
	ReadStatus(ctx context.Context, userID string) (map[string]time.Time, error)

	// Conversations.
	CreateConversation(ctx context.Context, c model.Conversation) (model.Conversation, error)
	ListConversations(ctx context.Context, userID string) ([]model.Conversation, error)
	GetConversation(ctx context.Context, id string) (model.Conversation, error)
	UpdateConversationTitle(ctx context.Context, id, title string) (model.Conversation, error)
	DeleteConversation(ctx context.Context, id string) error
	CreateMessage(ctx context.Context, m model.Message) (model.Message, error)
	ListMessages(ctx context.Context, conversationID string) ([]model.Message, error)

	// Answer runs / events / answers.
	CreateRun(ctx context.Context, r model.AnswerRun) (model.AnswerRun, error)
	GetRun(ctx context.Context, id string) (model.AnswerRun, error)
	SetRunStatus(ctx context.Context, id, status, errorCode string) error
	AddRunEvent(ctx context.Context, runID, eventType string, payload any) error
	ListRunEvents(ctx context.Context, runID string, afterID int64) ([]model.RunEvent, error)
	CreateAnswer(ctx context.Context, a model.Answer, steps []model.AnswerStep, citations []model.Citation) (model.Answer, error)
	GetAnswerByRun(ctx context.Context, runID string) (model.Answer, []model.AnswerStep, []model.Citation, error)
	GetAnswer(ctx context.Context, id string) (model.Answer, error)
	ListAnswerSteps(ctx context.Context, answerID string) ([]model.AnswerStep, error)

	// Jobs.
	EnqueueJob(ctx context.Context, j model.Job) (model.Job, error)
	ClaimJobs(ctx context.Context, limit int, answers bool) ([]model.Job, error)
	RequeueStaleJobs(ctx context.Context, olderThan time.Duration) (int64, error)
	CompleteJob(ctx context.Context, id string) error
	FailJob(ctx context.Context, id, errCode string, retryDelay time.Duration) error
	ListJobs(ctx context.Context, status string, limit int) ([]model.Job, error)
	RetryJob(ctx context.Context, id string) error

	// Search.
	SearchChunks(ctx context.Context, query string, docIDs, categories []string, limit int) ([]model.SearchHit, error)

	// Document relations.
	ListRelations(ctx context.Context, docID string) ([]model.DocumentRelation, error)
	CreateRelation(ctx context.Context, rel model.DocumentRelation) (model.DocumentRelation, error)
	VerifyRelation(ctx context.Context, id, status string) error

	// Feedback / saved instructions / action routes.
	AddFeedback(ctx context.Context, answerID, userID, rating, comment string) error
	CreateSavedInstruction(ctx context.Context, si model.SavedInstruction) (model.SavedInstruction, error)
	ListSavedInstructions(ctx context.Context, userID string) ([]model.SavedInstruction, error)
	DeleteSavedInstruction(ctx context.Context, id, userID string) error
	CreateActionRoute(ctx context.Context, route model.ActionRoute, steps []model.ActionRouteStep) (model.ActionRoute, error)
	ListActionRoutes(ctx context.Context, userID string) ([]model.ActionRoute, error)
	GetActionRoute(ctx context.Context, id string) (model.ActionRoute, []model.ActionRouteStep, error)
	SetRouteStepDone(ctx context.Context, routeID, stepID string, done bool) error

	// Lookup for relation auto-detection: finds a document by its number.
	FindDocumentByNumber(ctx context.Context, number string) (model.Document, error)

	// Onboarding: folders/visibility, departments, user organisation,
	// assignments, plans, notifications (onboarding.go).
	UpdateDocumentAdmin(ctx context.Context, id string, title, folder, visibility *string) error
	CreateDepartment(ctx context.Context, name string) (model.Department, error)
	RenameDepartment(ctx context.Context, id, name string) error
	DeleteDepartment(ctx context.Context, id string) error
	DepartmentByName(ctx context.Context, name string) (model.Department, error)
	ListUserProfiles(ctx context.Context) ([]model.UserProfile, error)
	GetUserProfile(ctx context.Context, id string) (model.UserProfile, error)
	UpdateUserOrg(ctx context.Context, id string, role, departmentID, position *string, active *bool) error
	IsUserActive(ctx context.Context, id string) (bool, error)
	ListAssignments(ctx context.Context, documentID, targetType, targetID string) ([]model.Assignment, error)
	UpsertAssignment(ctx context.Context, a model.Assignment) (model.Assignment, bool, error)
	DeleteAssignment(ctx context.Context, id string) error
	CountAssignments(ctx context.Context) (int, error)
	UsersForTarget(ctx context.Context, targetType, targetID string) ([]string, error)
	PlanFor(ctx context.Context, userID string) ([]model.PlanItem, error)
	CreateNotifications(ctx context.Context, userIDs []string, kind, documentID, title, body string) error
	ListNotifications(ctx context.Context, userID string, limit int) ([]model.Notification, int, error)
	MarkNotificationsRead(ctx context.Context, userID, id string) error

	// Audit.
	AddAuditLog(ctx context.Context, userID, entity, entityID, action, reason string, oldVal, newVal any) error
}
