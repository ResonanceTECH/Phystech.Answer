package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/parse"
)

// Postgres is a Store backed by PostgreSQL. documents carry the projected
// content of their latest version; the versioned tables are the source of
// truth for structure and citations.
type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ---------- documents ----------

const documentProjection = `
SELECT d.id, d.title, d.document_number, d.document_type, d.category,
       d.document_date, d.access_level, d.tags, d.created_at, d.updated_at,
       COALESCE(v.version_number, 0), COALESCE(v.content, ''),
       COALESCE(v.legal_status, ''), COALESCE(v.processing_status, ''),
       d.visibility
FROM documents d
LEFT JOIN LATERAL (
    SELECT version_number, content, legal_status, processing_status
    FROM document_versions
    WHERE document_id = d.id
    ORDER BY (processing_status = 'published') DESC, version_number DESC
    LIMIT 1
) v ON true`

func scanDocument(row pgx.Row) (model.Document, error) {
	var d model.Document
	var num, typ, cat, access, legal, proc *string
	err := row.Scan(&d.ID, &d.Title, &num, &typ, &cat,
		&d.DocumentDate, &access, &d.Tags, &d.CreatedAt, &d.UpdatedAt,
		&d.Version, &d.Content, &legal, &proc, &d.Visibility)
	if err != nil {
		return model.Document{}, err
	}
	d.DocumentNumber = strVal(num)
	d.DocumentType = strVal(typ)
	d.Category = strVal(cat)
	d.AccessLevel = strVal(access)
	d.LegalStatus = strVal(legal)
	d.Processing = strVal(proc)
	if d.Tags == nil {
		d.Tags = []string{}
	}
	return d, nil
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (p *Postgres) ListDocuments(ctx context.Context) ([]model.Document, error) {
	rows, err := p.pool.Query(ctx, documentProjection+` ORDER BY d.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []model.Document
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (p *Postgres) GetDocument(ctx context.Context, id string) (model.Document, error) {
	d, err := scanDocument(p.pool.QueryRow(ctx, documentProjection+` WHERE d.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Document{}, ErrNotFound
	}
	return d, err
}

// CreateDocument inserts the document plus version 1 with derived
// sections and chunks in one transaction.
func (p *Postgres) CreateDocument(ctx context.Context, doc model.Document) (model.Document, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.Document{}, err
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	doc.ID = newID("doc")
	doc.CreatedAt, doc.UpdatedAt = now, now
	if doc.Tags == nil {
		doc.Tags = []string{}
	}
	_, err = tx.Exec(ctx, `INSERT INTO documents
		(id, title, document_number, document_type, category, access_level, tags, created_at, updated_at, document_date, visibility)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		doc.ID, doc.Title, doc.DocumentNumber, doc.DocumentType, doc.Category,
		defaultStr(doc.AccessLevel, "internal"), doc.Tags, now, now, doc.DocumentDate,
		defaultStr(doc.Visibility, "active"))
	if err != nil {
		return model.Document{}, err
	}

	v, err := p.insertVersionTx(ctx, tx, model.DocumentVersion{
		DocumentID: doc.ID, SourceFileID: doc.SourceFileID,
		VersionNumber: 1, Content: doc.Content,
		ProcessingStatus: "published", LegalStatus: "in_force", ReviewStatus: "approved",
	})
	if err != nil {
		return model.Document{}, err
	}
	if err := p.rebuildStructureTx(ctx, tx, v.ID, doc.Content); err != nil {
		return model.Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Document{}, err
	}
	doc.Version = 1
	doc.Processing = "published"
	doc.LegalStatus = "in_force"
	return doc, nil
}

// UpdateDocument updates metadata and stores the new content as a new
// version (version_number = max+1).
func (p *Postgres) UpdateDocument(ctx context.Context, doc model.Document) (model.Document, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.Document{}, err
	}
	defer tx.Rollback(ctx)

	var maxVer int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(version_number),0) FROM document_versions WHERE document_id=$1`,
		doc.ID).Scan(&maxVer)
	if err != nil {
		return model.Document{}, err
	}
	res, err := tx.Exec(ctx, `UPDATE documents
		SET title=$2, document_number=$3, document_type=$4, category=$5, tags=$6, updated_at=now()
		WHERE id=$1`,
		doc.ID, doc.Title, doc.DocumentNumber, doc.DocumentType, doc.Category, doc.Tags)
	if err != nil {
		return model.Document{}, err
	}
	if res.RowsAffected() == 0 {
		return model.Document{}, ErrNotFound
	}
	v, err := p.insertVersionTx(ctx, tx, model.DocumentVersion{
		DocumentID: doc.ID, VersionNumber: maxVer + 1, Content: doc.Content,
		ProcessingStatus: "published", LegalStatus: "in_force", ReviewStatus: "approved",
	})
	if err != nil {
		return model.Document{}, err
	}
	if err := p.rebuildStructureTx(ctx, tx, v.ID, doc.Content); err != nil {
		return model.Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Document{}, err
	}
	return p.GetDocument(ctx, doc.ID)
}

func (p *Postgres) DeleteDocument(ctx context.Context, id string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT DISTINCT source_file_id
		FROM document_versions
		WHERE document_id=$1 AND source_file_id IS NOT NULL`, id)
	if err != nil {
		return err
	}
	var sourceFileIDs []string
	for rows.Next() {
		var fileID string
		if err := rows.Scan(&fileID); err != nil {
			rows.Close()
			return err
		}
		sourceFileIDs = append(sourceFileIDs, fileID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	res, err := tx.Exec(ctx, `DELETE FROM documents WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	for _, fileID := range sourceFileIDs {
		if _, err := tx.Exec(ctx, `DELETE FROM source_files sf
			WHERE sf.id=$1
			  AND NOT EXISTS (SELECT 1 FROM document_versions v WHERE v.source_file_id=sf.id)`, fileID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// nullStr для nullable-колонок: пустая строка → SQL NULL.
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (p *Postgres) insertVersionTx(ctx context.Context, tx pgx.Tx, v model.DocumentVersion) (model.DocumentVersion, error) {
	v.ID = newID("ver")
	now := time.Now()
	v.CreatedAt, v.UpdatedAt = now, now
	err := tx.QueryRow(ctx, `INSERT INTO document_versions
		(id, document_id, source_file_id, version_number, content,
		 effective_from, effective_to, legal_status, review_status,
		 processing_status, published_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		        CASE WHEN $10='published' THEN now() ELSE NULL END, $11, $11)
		RETURNING published_at`,
		v.ID, v.DocumentID, nilIfEmpty(v.SourceFileID), v.VersionNumber, v.Content,
		v.EffectiveFrom, v.EffectiveTo,
		defaultStr(v.LegalStatus, "in_force"), defaultStr(v.ReviewStatus, "approved"),
		defaultStr(v.ProcessingStatus, "published"), now).Scan(&v.PublishedAt)
	if err != nil {
		return model.DocumentVersion{}, err
	}
	return v, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// rebuildStructureTx replaces sections and chunks for a version by
// splitting the content into numbered пункты.
func (p *Postgres) rebuildStructureTx(ctx context.Context, tx pgx.Tx, versionID, content string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM document_sections WHERE version_id=$1`, versionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM document_chunks WHERE version_id=$1`, versionID); err != nil {
		return err
	}
	for _, sec := range parse.SplitSections(content) {
		secID := newID("sec")
		if _, err := tx.Exec(ctx, `INSERT INTO document_sections
			(id, version_id, section_number, title, content, position)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			secID, versionID, sec.Number, sec.Title, sec.Content, sec.Position); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO document_chunks
			(id, version_id, section_id, content, chunk_index)
			VALUES ($1,$2,$3,$4,$5)`,
			newID("chk"), versionID, secID, sec.Content, sec.Position); err != nil {
			return err
		}
	}
	return nil
}

// ---------- versions / structure ----------

func scanVersion(row pgx.Row) (model.DocumentVersion, error) {
	var v model.DocumentVersion
	var srcID *string
	err := row.Scan(&v.ID, &v.DocumentID, &srcID, &v.VersionNumber, &v.Content,
		&v.EffectiveFrom, &v.EffectiveTo, &v.LegalStatus, &v.ReviewStatus,
		&v.ProcessingStatus, &v.ProcessingError, &v.PublishedAt,
		&v.CreatedAt, &v.UpdatedAt)
	v.SourceFileID = strVal(srcID)
	return v, err
}

const versionCols = `id, document_id, source_file_id, version_number, content,
	effective_from, effective_to, legal_status, review_status,
	processing_status, processing_error, published_at, created_at, updated_at`

func (p *Postgres) ListVersions(ctx context.Context, docID string) ([]model.DocumentVersion, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT `+versionCols+` FROM document_versions WHERE document_id=$1 ORDER BY version_number`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DocumentVersion
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (p *Postgres) GetVersion(ctx context.Context, versionID string) (model.DocumentVersion, error) {
	v, err := scanVersion(p.pool.QueryRow(ctx,
		`SELECT `+versionCols+` FROM document_versions WHERE id=$1`, versionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.DocumentVersion{}, ErrNotFound
	}
	return v, err
}

func (p *Postgres) CreateVersion(ctx context.Context, v model.DocumentVersion) (model.DocumentVersion, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.DocumentVersion{}, err
	}
	defer tx.Rollback(ctx)
	if v.VersionNumber <= 0 {
		var maxVer int
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(MAX(version_number),0) FROM document_versions WHERE document_id=$1`,
			v.DocumentID).Scan(&maxVer); err != nil {
			return model.DocumentVersion{}, err
		}
		v.VersionNumber = maxVer + 1
	}
	out, err := p.insertVersionTx(ctx, tx, v)
	if err != nil {
		return model.DocumentVersion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.DocumentVersion{}, err
	}
	return out, nil
}

func (p *Postgres) ListSections(ctx context.Context, versionID string) ([]model.Section, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, version_id, COALESCE(parent_id,''),
		section_number, title, content, position
		FROM document_sections WHERE version_id=$1 ORDER BY position`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Section
	for rows.Next() {
		var s model.Section
		if err := rows.Scan(&s.ID, &s.VersionID, &s.ParentID,
			&s.SectionNumber, &s.Title, &s.Content, &s.Position); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (p *Postgres) ListChunks(ctx context.Context, versionID string) ([]model.Chunk, error) {
	rows, err := p.pool.Query(ctx, `SELECT c.id, c.version_id, COALESCE(c.section_id,''),
		c.content, c.chunk_index, c.location_type, c.page_number,
		c.sheet_name, c.cell_range, COALESCE(s.section_number,'')
		FROM document_chunks c
		LEFT JOIN document_sections s ON s.id = c.section_id
		WHERE c.version_id=$1 ORDER BY c.chunk_index`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Chunk
	for rows.Next() {
		var c model.Chunk
		if err := rows.Scan(&c.ID, &c.VersionID, &c.SectionID,
			&c.Content, &c.ChunkIndex, &c.LocationType, &c.PageNumber,
			&c.SheetName, &c.CellRange, &c.SectionNumber); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateDocumentMeta fills normative metadata detected while parsing.
// Empty values do not overwrite existing ones.
func (p *Postgres) UpdateDocumentMeta(ctx context.Context, id, number, docType string, date *time.Time) error {
	_, err := p.pool.Exec(ctx, `UPDATE documents SET
		document_number = CASE WHEN $2 <> '' THEN $2 ELSE document_number END,
		document_type   = CASE WHEN $3 <> '' THEN $3 ELSE document_type END,
		document_date   = COALESCE($4, document_date),
		updated_at = now()
		WHERE id = $1`, id, number, docType, date)
	return err
}

func (p *Postgres) SetVersionContent(ctx context.Context, versionID, content string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	res, err := tx.Exec(ctx,
		`UPDATE document_versions SET content=$2, updated_at=now() WHERE id=$1`,
		versionID, content)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := p.rebuildStructureTx(ctx, tx, versionID, content); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *Postgres) SetVersionStatus(ctx context.Context, versionID, processingStatus, processingError string) error {
	res, err := p.pool.Exec(ctx, `UPDATE document_versions
		SET processing_status=$2, processing_error=$3,
		    published_at = CASE WHEN $2='published' AND published_at IS NULL THEN now() ELSE published_at END,
		    updated_at=now()
		WHERE id=$1`, versionID, processingStatus, processingError)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- source files ----------

func (p *Postgres) CreateSourceFile(ctx context.Context, f model.SourceFile) (model.SourceFile, error) {
	f.ID = newID("file")
	f.UploadedAt = time.Now()
	err := p.pool.QueryRow(ctx, `INSERT INTO source_files
		(id, original_filename, storage_key, mime_type, size_bytes, sha256, uploaded_by, uploaded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING uploaded_at`,
		f.ID, f.OriginalFilename, f.StorageKey, f.MimeType, f.SizeBytes,
		f.SHA256, nilIfEmpty(f.UploadedBy), f.UploadedAt).Scan(&f.UploadedAt)
	if isUniqueViolation(err) {
		return model.SourceFile{}, ErrConflict
	}
	return f, err
}

func (p *Postgres) GetSourceFileBySHA(ctx context.Context, sha string) (model.SourceFile, error) {
	var f model.SourceFile
	err := p.pool.QueryRow(ctx, `SELECT id, original_filename, storage_key, mime_type,
		size_bytes, sha256, COALESCE(uploaded_by,''), uploaded_at
		FROM source_files WHERE sha256=$1`, sha).Scan(
		&f.ID, &f.OriginalFilename, &f.StorageKey, &f.MimeType,
		&f.SizeBytes, &f.SHA256, &f.UploadedBy, &f.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SourceFile{}, ErrNotFound
	}
	return f, err
}

func (p *Postgres) GetSourceFile(ctx context.Context, id string) (model.SourceFile, error) {
	var f model.SourceFile
	err := p.pool.QueryRow(ctx, `SELECT id, original_filename, storage_key, mime_type,
		size_bytes, sha256, COALESCE(uploaded_by,''), uploaded_at
		FROM source_files WHERE id=$1`, id).Scan(
		&f.ID, &f.OriginalFilename, &f.StorageKey, &f.MimeType,
		&f.SizeBytes, &f.SHA256, &f.UploadedBy, &f.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SourceFile{}, ErrNotFound
	}
	return f, err
}

// ---------- users / auth ----------

const userCols = `id, name, first_name, last_name, email, COALESCE(login, ''), role,
	employee_category, COALESCE(department_id, ''), "position",
	onboarding_complete, email_verified_at, created_at`

func scanUser(row pgx.Row) (model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Name, &u.FirstName, &u.LastName, &u.Email, &u.Login,
		&u.Role, &u.EmployeeCategory, &u.DepartmentID, &u.Position,
		&u.OnboardingComplete, &u.EmailVerifiedAt, &u.CreatedAt)
	return u, err
}

func (p *Postgres) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT `+userCols+` FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (p *Postgres) CreateUser(ctx context.Context, u model.User, passwordHash string) (model.User, error) {
	u.ID = newID("usr")
	u.CreatedAt = time.Now()
	if u.Name == "" {
		u.Name = strings.TrimSpace(u.FirstName + " " + u.LastName)
	}
	_, err := p.pool.Exec(ctx, `INSERT INTO users
		(id, name, first_name, last_name, email, login, password_hash, role,
		 employee_category, department_id, "position", onboarding_complete, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		u.ID, u.Name, u.FirstName, u.LastName, u.Email, nullStr(u.Login), passwordHash,
		defaultStr(u.Role, "employee"), u.EmployeeCategory,
		nullStr(u.DepartmentID), u.Position, u.OnboardingComplete, u.CreatedAt)
	if isUniqueViolation(err) {
		return model.User{}, ErrConflict
	}
	return u, err
}

func (p *Postgres) GetUser(ctx context.Context, id string) (model.User, error) {
	u, err := scanUser(p.pool.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	return u, err
}

func (p *Postgres) GetUserByLogin(ctx context.Context, login string) (model.User, string, error) {
	var u model.User
	var hash string
	err := p.pool.QueryRow(ctx, `SELECT `+userCols+`, password_hash
		FROM users WHERE login=$1`, login).Scan(
		&u.ID, &u.Name, &u.FirstName, &u.LastName, &u.Email, &u.Login,
		&u.Role, &u.EmployeeCategory, &u.DepartmentID, &u.Position,
		&u.OnboardingComplete, &u.EmailVerifiedAt, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, "", ErrNotFound
	}
	return u, hash, err
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (model.User, string, error) {
	var u model.User
	var hash string
	err := p.pool.QueryRow(ctx, `SELECT `+userCols+`, password_hash
		FROM users WHERE lower(email)=lower($1)`, email).Scan(
		&u.ID, &u.Name, &u.FirstName, &u.LastName, &u.Email, &u.Login,
		&u.Role, &u.EmployeeCategory, &u.DepartmentID, &u.Position,
		&u.OnboardingComplete, &u.EmailVerifiedAt, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, "", ErrNotFound
	}
	return u, hash, err
}

func (p *Postgres) UpdateUserProfile(ctx context.Context, id string, patch model.ProfilePatch) (model.User, error) {
	sets := []string{"updated_at = now()"}
	args := []any{id}
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if patch.FirstName != nil {
		add("first_name", strings.TrimSpace(*patch.FirstName))
	}
	if patch.LastName != nil {
		add("last_name", strings.TrimSpace(*patch.LastName))
	}
	if patch.EmployeeCategory != nil {
		add("employee_category", *patch.EmployeeCategory)
	}
	if patch.DepartmentID != nil {
		add("department_id", nullStr(*patch.DepartmentID))
	}
	if patch.Position != nil {
		add("position", *patch.Position)
	}
	if patch.OnboardingComplete != nil {
		add("onboarding_complete", *patch.OnboardingComplete)
	}
	u, err := scanUser(p.pool.QueryRow(ctx,
		`UPDATE users SET `+strings.Join(sets, ", ")+` WHERE id=$1 RETURNING `+userCols,
		args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, err
	}
	// name — производное поле; в одном UPDATE оно увидело бы старые значения.
	if patch.FirstName != nil || patch.LastName != nil {
		if _, err := p.pool.Exec(ctx,
			`UPDATE users SET name = btrim(concat_ws(' ', first_name, last_name)) WHERE id=$1`, id); err != nil {
			return model.User{}, err
		}
		u.Name = strings.TrimSpace(u.FirstName + " " + u.LastName)
	}
	return u, nil
}

func (p *Postgres) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	tag, err := p.pool.Exec(ctx,
		`UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, id, passwordHash)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (p *Postgres) CreateSession(ctx context.Context, userID, tokenHash string, ttl time.Duration) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO sessions (id, user_id, token_hash, last_seen_at, expires_at)
		VALUES ($1,$2,$3,now(),$4)`, newID("ses"), userID, tokenHash, time.Now().Add(ttl))
	return err
}

// userColsU — userCols с алиасом u. для JOIN-запросов (id/created_at
// иначе неоднозначны между sessions и users).
const userColsU = `u.id, u.name, u.first_name, u.last_name, u.email,
	COALESCE(u.login, ''), u.role, u.employee_category,
	COALESCE(u.department_id, ''), u."position",
	u.onboarding_complete, u.email_verified_at, u.created_at`

func (p *Postgres) GetSessionUser(ctx context.Context, tokenHash string) (model.User, error) {
	u, err := scanUser(p.pool.QueryRow(ctx, `SELECT `+userColsU+`
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash=$1 AND s.expires_at > now() AND s.revoked_at IS NULL`, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err == nil {
		_, _ = p.pool.Exec(ctx, `UPDATE sessions SET last_seen_at=now() WHERE token_hash=$1`, tokenHash)
	}
	return u, err
}

func (p *Postgres) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at=now() WHERE token_hash=$1`, tokenHash)
	return err
}

func (p *Postgres) RevokeUserSessions(ctx context.Context, userID string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}

func (p *Postgres) CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, ttl time.Duration) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1,$2,$3,$4)`, newID("prt"), userID, tokenHash, time.Now().Add(ttl))
	return err
}

func (p *Postgres) ConsumePasswordResetToken(ctx context.Context, tokenHash string) (model.User, error) {
	var id, userID string
	var usedAt *time.Time
	var expiresAt time.Time
	err := p.pool.QueryRow(ctx,
		`SELECT id, user_id, used_at, expires_at FROM password_reset_tokens WHERE token_hash=$1`,
		tokenHash).Scan(&id, &userID, &usedAt, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, err
	}
	if usedAt != nil {
		return model.User{}, ErrTokenUsed
	}
	if time.Now().After(expiresAt) {
		return model.User{}, ErrTokenExpired
	}
	// Одноразовость гонко-устойчива: второй параллельный UPDATE ничего не найдёт.
	tag, err := p.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at=now() WHERE id=$1 AND used_at IS NULL`, id)
	if err != nil {
		return model.User{}, err
	}
	if tag.RowsAffected() == 0 {
		return model.User{}, ErrTokenUsed
	}
	return p.GetUser(ctx, userID)
}

// ---------- read marks ----------

func (p *Postgres) MarkRead(ctx context.Context, docID, userID string) (time.Time, error) {
	var t time.Time
	err := p.pool.QueryRow(ctx, `INSERT INTO user_doc_reads (user_id, document_id, read_at)
		VALUES ($1,$2,now())
		ON CONFLICT (user_id, document_id) DO UPDATE SET read_at=now()
		RETURNING read_at`, userID, docID).Scan(&t)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return time.Time{}, ErrNotFound
		}
		return time.Time{}, err
	}
	return t, nil
}

func (p *Postgres) ReadStatus(ctx context.Context, userID string) (map[string]time.Time, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT document_id, read_at FROM user_doc_reads WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]time.Time{}
	for rows.Next() {
		var id string
		var t time.Time
		if err := rows.Scan(&id, &t); err != nil {
			return nil, err
		}
		out[id] = t
	}
	return out, rows.Err()
}

// ---------- conversations ----------

func (p *Postgres) CreateConversation(ctx context.Context, c model.Conversation) (model.Conversation, error) {
	c.ID = newID("conv")
	now := time.Now()
	c.CreatedAt, c.UpdatedAt = now, now
	_, err := p.pool.Exec(ctx, `INSERT INTO conversations (id, user_id, title, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$4)`, c.ID, nilIfEmpty(c.UserID), c.Title, now)
	return c, err
}

func (p *Postgres) ListConversations(ctx context.Context, userID string) ([]model.Conversation, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, COALESCE(user_id,''), title, created_at, updated_at
		FROM conversations WHERE user_id=$1 OR ($1='' AND user_id IS NULL)
		ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Conversation
	for rows.Next() {
		var c model.Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (p *Postgres) GetConversation(ctx context.Context, id string) (model.Conversation, error) {
	var c model.Conversation
	err := p.pool.QueryRow(ctx, `SELECT id, COALESCE(user_id,''), title, created_at, updated_at
		FROM conversations WHERE id=$1`, id).Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Conversation{}, ErrNotFound
	}
	return c, err
}

func (p *Postgres) UpdateConversationTitle(ctx context.Context, id, title string) (model.Conversation, error) {
	res, err := p.pool.Exec(ctx,
		`UPDATE conversations SET title=$2, updated_at=now() WHERE id=$1`, id, title)
	if err != nil {
		return model.Conversation{}, err
	}
	if res.RowsAffected() == 0 {
		return model.Conversation{}, ErrNotFound
	}
	return p.GetConversation(ctx, id)
}

func (p *Postgres) DeleteConversation(ctx context.Context, id string) error {
	res, err := p.pool.Exec(ctx, `DELETE FROM conversations WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) CreateMessage(ctx context.Context, m model.Message) (model.Message, error) {
	m.ID = newID("msg")
	m.CreatedAt = time.Now()
	_, err := p.pool.Exec(ctx, `INSERT INTO messages (id, conversation_id, role, content, created_at)
		VALUES ($1,$2,$3,$4,$5)`, m.ID, m.ConversationID, m.Role, m.Content, m.CreatedAt)
	if err == nil {
		p.pool.Exec(ctx, `UPDATE conversations SET updated_at=now() WHERE id=$1`, m.ConversationID)
	}
	return m, err
}

func (p *Postgres) ListMessages(ctx context.Context, conversationID string) ([]model.Message, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, conversation_id, role, content, created_at
		FROM messages WHERE conversation_id=$1 ORDER BY created_at`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ---------- answer runs / events / answers ----------

func (p *Postgres) CreateRun(ctx context.Context, r model.AnswerRun) (model.AnswerRun, error) {
	r.ID = newID("run")
	r.Status = "queued"
	r.CreatedAt = time.Now()
	_, err := p.pool.Exec(ctx, `INSERT INTO answer_runs (id, conversation_id, message_id, status, created_at)
		VALUES ($1,$2,$3,$4,$5)`, r.ID, r.ConversationID, nilIfEmpty(r.MessageID), r.Status, r.CreatedAt)
	return r, err
}

func (p *Postgres) GetRun(ctx context.Context, id string) (model.AnswerRun, error) {
	var r model.AnswerRun
	err := p.pool.QueryRow(ctx, `SELECT id, conversation_id, COALESCE(message_id,''),
		status, error_code, started_at, completed_at, created_at
		FROM answer_runs WHERE id=$1`, id).Scan(
		&r.ID, &r.ConversationID, &r.MessageID, &r.Status, &r.ErrorCode,
		&r.StartedAt, &r.CompletedAt, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.AnswerRun{}, ErrNotFound
	}
	return r, err
}

func (p *Postgres) SetRunStatus(ctx context.Context, id, status, errorCode string) error {
	_, err := p.pool.Exec(ctx, `UPDATE answer_runs SET status=$2, error_code=$3,
		started_at = CASE WHEN $2='running' AND started_at IS NULL THEN now() ELSE started_at END,
		completed_at = CASE WHEN $2 IN ('completed','failed','cancelled') THEN now() ELSE completed_at END
		WHERE id=$1`, id, status, errorCode)
	return err
}

func (p *Postgres) AddRunEvent(ctx context.Context, runID, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `INSERT INTO run_events (run_id, event_type, payload)
		VALUES ($1,$2,$3)`, runID, eventType, data)
	return err
}

func (p *Postgres) ListRunEvents(ctx context.Context, runID string, afterID int64) ([]model.RunEvent, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, run_id, event_type, payload, created_at
		FROM run_events WHERE run_id=$1 AND id>$2 ORDER BY id`, runID, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.RunEvent
	for rows.Next() {
		var e model.RunEvent
		if err := rows.Scan(&e.ID, &e.RunID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CreateAnswer inserts the answer, its steps and citations atomically.
// citations[].StepOrder matches the step's Order field; 0 means unbound.
func (p *Postgres) CreateAnswer(ctx context.Context, a model.Answer, steps []model.AnswerStep, citations []model.Citation) (model.Answer, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.Answer{}, err
	}
	defer tx.Rollback(ctx)

	a.ID = newID("ans")
	a.CreatedAt = time.Now()
	if a.Status == "" {
		a.Status = "answered"
	}
	if a.Warnings == nil {
		a.Warnings = []string{}
	}
	if a.MissingInformation == nil {
		a.MissingInformation = []string{}
	}
	_, err = tx.Exec(ctx, `INSERT INTO answers (id, run_id, status, summary, warnings, missing_information, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.RunID, a.Status, a.Summary, a.Warnings, a.MissingInformation, a.CreatedAt)
	if err != nil {
		return model.Answer{}, err
	}

	stepIDByOrder := map[int]string{}
	for _, s := range steps {
		s.ID = newID("step")
		s.AnswerID = a.ID
		if _, err := tx.Exec(ctx, `INSERT INTO answer_steps
			(id, answer_id, step_order, title, description, responsible, deadline, condition_text, required)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			s.ID, a.ID, s.Order, s.Title, s.Description,
			s.Responsible, s.Deadline, s.Condition, s.Required); err != nil {
			return model.Answer{}, err
		}
		stepIDByOrder[s.Order] = s.ID
	}

	for i, c := range citations {
		// Readers ORDER BY id: keep the ML source order so [n] markers stay valid.
		c.ID = fmt.Sprintf("cit_%s_%03d", a.ID, i)
		stepID := ""
		if c.StepOrder > 0 {
			stepID = stepIDByOrder[c.StepOrder]
		}
		if _, err := tx.Exec(ctx, `INSERT INTO answer_citations
			(id, answer_id, step_id, chunk_id, document_version_id, document_id,
			 document_title, section_number, page_number, quoted_text, score)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			c.ID, a.ID, nilIfEmpty(stepID), nilIfEmpty(c.ChunkID),
			nilIfEmpty(c.DocumentVersionID), nilIfEmpty(c.DocumentID),
			c.DocumentTitle, c.SectionNumber, c.PageNumber, c.Quote, c.Score); err != nil {
			return model.Answer{}, err
		}
		citations[i].ID = c.ID
	}
	return a, tx.Commit(ctx)
}

func (p *Postgres) GetAnswerByRun(ctx context.Context, runID string) (model.Answer, []model.AnswerStep, []model.Citation, error) {
	var a model.Answer
	err := p.pool.QueryRow(ctx, `SELECT a.id, a.run_id, COALESCE(r.conversation_id,''),
		a.status, a.summary, a.warnings, a.missing_information, a.created_at
		FROM answers a JOIN answer_runs r ON r.id = a.run_id
		WHERE a.run_id=$1`, runID).Scan(
		&a.ID, &a.RunID, &a.ConversationID, &a.Status, &a.Summary,
		&a.Warnings, &a.MissingInformation, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Answer{}, nil, nil, ErrNotFound
	}
	if err != nil {
		return model.Answer{}, nil, nil, err
	}

	stepRows, err := p.pool.Query(ctx, `SELECT id, step_order, title, description,
		responsible, deadline, condition_text, required
		FROM answer_steps WHERE answer_id=$1 ORDER BY step_order`, a.ID)
	if err != nil {
		return a, nil, nil, err
	}
	var steps []model.AnswerStep
	for stepRows.Next() {
		var s model.AnswerStep
		if err := stepRows.Scan(&s.ID, &s.Order, &s.Title, &s.Description,
			&s.Responsible, &s.Deadline, &s.Condition, &s.Required); err != nil {
			stepRows.Close()
			return a, nil, nil, err
		}
		s.CitationIDs = []string{}
		steps = append(steps, s)
	}
	stepRows.Close()

	citRows, err := p.pool.Query(ctx, `SELECT id, COALESCE(step_id,''), COALESCE(chunk_id,''),
		COALESCE(document_version_id,''), COALESCE(document_id,''),
		document_title, section_number, page_number, quoted_text, score
		FROM answer_citations WHERE answer_id=$1 ORDER BY id`, a.ID)
	if err != nil {
		return a, steps, nil, err
	}
	var citations []model.Citation
	for citRows.Next() {
		var c model.Citation
		if err := citRows.Scan(&c.ID, &c.StepID, &c.ChunkID,
			&c.DocumentVersionID, &c.DocumentID,
			&c.DocumentTitle, &c.SectionNumber, &c.PageNumber, &c.Quote, &c.Score); err != nil {
			citRows.Close()
			return a, steps, nil, err
		}
		citations = append(citations, c)
	}
	citRows.Close()

	stepByID := map[string]int{}
	for i := range steps {
		stepByID[steps[i].ID] = i
	}
	for _, c := range citations {
		if i, ok := stepByID[c.StepID]; ok {
			steps[i].CitationIDs = append(steps[i].CitationIDs, c.ID)
		}
	}
	return a, steps, citations, nil
}

// GetAnswer returns the answer row joined with its run's conversation id.
func (p *Postgres) GetAnswer(ctx context.Context, id string) (model.Answer, error) {
	var a model.Answer
	err := p.pool.QueryRow(ctx, `SELECT a.id, a.run_id, COALESCE(r.conversation_id,''),
		a.status, a.summary, a.warnings, a.missing_information, a.created_at
		FROM answers a JOIN answer_runs r ON r.id = a.run_id
		WHERE a.id=$1`, id).Scan(
		&a.ID, &a.RunID, &a.ConversationID, &a.Status, &a.Summary,
		&a.Warnings, &a.MissingInformation, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Answer{}, ErrNotFound
	}
	return a, err
}

func (p *Postgres) ListAnswerSteps(ctx context.Context, answerID string) ([]model.AnswerStep, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, step_order, title, description,
		responsible, deadline, condition_text, required
		FROM answer_steps WHERE answer_id=$1 ORDER BY step_order`, answerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var steps []model.AnswerStep
	for rows.Next() {
		var s model.AnswerStep
		if err := rows.Scan(&s.ID, &s.Order, &s.Title, &s.Description,
			&s.Responsible, &s.Deadline, &s.Condition, &s.Required); err != nil {
			return nil, err
		}
		s.CitationIDs = []string{}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

// ---------- jobs ----------

func (p *Postgres) EnqueueJob(ctx context.Context, j model.Job) (model.Job, error) {
	j.ID = newID("job")
	j.Status = "queued"
	j.CreatedAt = time.Now()
	payload := j.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	err := p.pool.QueryRow(ctx, `INSERT INTO processing_jobs
		(id, document_version_id, job_type, status, payload, max_attempts, idempotency_key, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT DO NOTHING
		RETURNING id`, j.ID, nilIfEmpty(j.DocumentVersionID), j.JobType, j.Status,
		payload, defaultInt(j.MaxAttempts, 3), nilIfEmpty(j.IdempotencyKey), j.CreatedAt).Scan(&j.ID)
	if errors.Is(err, pgx.ErrNoRows) && j.IdempotencyKey != "" {
		// Идемпотентный повтор: возвращаем существующую задачу.
		var existing model.Job
		qerr := p.pool.QueryRow(ctx, `SELECT id, COALESCE(document_version_id,''), job_type, status,
			payload, progress, attempts, max_attempts, retry_at, error_code,
			COALESCE(idempotency_key,''), created_at, completed_at
			FROM processing_jobs WHERE idempotency_key=$1`, j.IdempotencyKey).Scan(
			&existing.ID, &existing.DocumentVersionID, &existing.JobType, &existing.Status,
			&existing.Payload, &existing.Progress, &existing.Attempts, &existing.MaxAttempts,
			&existing.RetryAt, &existing.ErrorCode, &existing.IdempotencyKey,
			&existing.CreatedAt, &existing.CompletedAt)
		return existing, qerr
	}
	return j, err
}

func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// ClaimJobs atomically picks queued jobs via FOR UPDATE SKIP LOCKED and
// marks them running.
func (p *Postgres) ClaimJobs(ctx context.Context, limit int) ([]model.Job, error) {
	rows, err := p.pool.Query(ctx, `UPDATE processing_jobs
		SET status='running', locked_at=now(), attempts=attempts+1
		WHERE id IN (
			SELECT id FROM processing_jobs
			WHERE status='queued' AND (retry_at IS NULL OR retry_at <= now())
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, COALESCE(document_version_id,''), job_type, status,
			payload, progress, attempts, max_attempts, retry_at, error_code,
			COALESCE(idempotency_key,''), created_at, completed_at`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

// RequeueStaleJobs returns jobs whose worker died mid-flight back to the
// queue.
func (p *Postgres) RequeueStaleJobs(ctx context.Context, olderThan time.Duration) (int64, error) {
	res, err := p.pool.Exec(ctx, `UPDATE processing_jobs
		SET status='queued', retry_at=now()
		WHERE status='running' AND locked_at < now() - $1::interval`,
		olderThan.String())
	return res.RowsAffected(), err
}

func (p *Postgres) CompleteJob(ctx context.Context, id string) error {
	_, err := p.pool.Exec(ctx, `UPDATE processing_jobs
		SET status='succeeded', progress=100, completed_at=now() WHERE id=$1`, id)
	return err
}

// FailJob marks the job failed permanently when attempts are exhausted,
// otherwise requeues it with a backoff delay.
func (p *Postgres) FailJob(ctx context.Context, id, errCode string, retryDelay time.Duration) error {
	_, err := p.pool.Exec(ctx, `UPDATE processing_jobs SET
		error_code=$2,
		status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'queued' END,
		retry_at = CASE WHEN attempts >= max_attempts THEN NULL ELSE now() + $3::interval END,
		completed_at = CASE WHEN attempts >= max_attempts THEN now() ELSE NULL END
		WHERE id=$1`, id, errCode, retryDelay.String())
	return err
}

func (p *Postgres) ListJobs(ctx context.Context, status string, limit int) ([]model.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, COALESCE(document_version_id,''), job_type, status,
		payload, progress, attempts, max_attempts, retry_at, error_code,
		COALESCE(idempotency_key,''), created_at, completed_at
		FROM processing_jobs`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT ` + itoa(limit)
	rows, err := p.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (p *Postgres) RetryJob(ctx context.Context, id string) error {
	res, err := p.pool.Exec(ctx, `UPDATE processing_jobs
		SET status='queued', attempts=0, retry_at=NULL, error_code='', progress=0
		WHERE id=$1 AND status IN ('failed','requires_review','cancelled')`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanJobs(rows pgx.Rows) ([]model.Job, error) {
	var out []model.Job
	for rows.Next() {
		var j model.Job
		if err := rows.Scan(&j.ID, &j.DocumentVersionID, &j.JobType, &j.Status,
			&j.Payload, &j.Progress, &j.Attempts, &j.MaxAttempts, &j.RetryAt,
			&j.ErrorCode, &j.IdempotencyKey, &j.CreatedAt, &j.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

// ---------- document relations ----------

func (p *Postgres) ListRelations(ctx context.Context, docID string) ([]model.DocumentRelation, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, source_version_id, COALESCE(target_document_id,''),
		COALESCE(target_version_id,''), relation_type, COALESCE(evidence_chunk_id,''),
		effective_from, review_status, created_at
		FROM document_relations WHERE target_document_id=$1 OR
		source_version_id IN (SELECT id FROM document_versions WHERE document_id=$1)`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DocumentRelation
	for rows.Next() {
		var r model.DocumentRelation
		if err := rows.Scan(&r.ID, &r.SourceVersionID, &r.TargetDocumentID,
			&r.TargetVersionID, &r.RelationType, &r.EvidenceChunkID,
			&r.EffectiveFrom, &r.ReviewStatus, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *Postgres) CreateRelation(ctx context.Context, r model.DocumentRelation) (model.DocumentRelation, error) {
	r.ID = newID("rel")
	r.CreatedAt = time.Now()
	_, err := p.pool.Exec(ctx, `INSERT INTO document_relations
		(id, source_version_id, target_document_id, target_version_id,
		 relation_type, evidence_chunk_id, effective_from, review_status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		r.ID, nilIfEmpty(r.SourceVersionID), nilIfEmpty(r.TargetDocumentID),
		nilIfEmpty(r.TargetVersionID), r.RelationType, nilIfEmpty(r.EvidenceChunkID),
		r.EffectiveFrom, defaultStr(r.ReviewStatus, "pending"), r.CreatedAt)
	return r, err
}

func (p *Postgres) VerifyRelation(ctx context.Context, id, status string) error {
	res, err := p.pool.Exec(ctx,
		`UPDATE document_relations SET review_status=$2 WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- search ----------

// SearchChunks runs Postgres full-text search (russian config) over
// published chunks, with optional document/category filters.
func (p *Postgres) SearchChunks(ctx context.Context, query string, docIDs, categories []string, limit int) ([]model.SearchHit, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	terms := strings.Fields(query)
	// Термы попадают в to_tsquery — выкидываем всё, что может сломать синтаксис.
	clean := terms[:0]
	for _, t := range terms {
		t = termSanitizer.ReplaceAllString(t, "")
		if len(t) > 0 {
			clean = append(clean, t)
		}
	}
	terms = clean
	if len(terms) == 0 {
		return nil, nil
	}
	// OR-соединение термов: plainto_tsquery даёт AND-семантику, которая для
	// русской морфологии слишком строга (разные словоформы — разные лексемы).
	// to_tsquery по 'a | b' стеммит каждый терм и даёт forgiving recall.
	q := `WITH q AS (
		SELECT to_tsquery('russian', string_agg(lower(t), ' | ')) AS tsq
		FROM unnest($1::text[]) t
	)
	SELECT c.id, c.version_id, d.id, d.title,
		COALESCE(s.section_number,''), c.content, v.legal_status,
		d.access_level,
		ts_rank(to_tsvector('russian', lower(c.content)), q.tsq) AS rank
		FROM document_chunks c
		CROSS JOIN q
		JOIN document_versions v ON v.id = c.version_id
		JOIN documents d ON d.id = v.document_id
		LEFT JOIN document_sections s ON s.id = c.section_id
		WHERE v.processing_status = 'published'
		  AND to_tsvector('russian', lower(c.content)) @@ q.tsq`
	args := []any{terms}
	n := 2
	if len(docIDs) > 0 {
		q += ` AND d.id = ANY($` + itoa(n) + `)`
		args = append(args, docIDs)
		n++
	}
	if len(categories) > 0 {
		q += ` AND d.category = ANY($` + itoa(n) + `)`
		args = append(args, categories)
		n++
	}
	q += ` ORDER BY rank DESC LIMIT ` + itoa(limit)
	rows, err := p.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SearchHit
	for rows.Next() {
		var h model.SearchHit
		if err := rows.Scan(&h.ChunkID, &h.VersionID, &h.DocumentID,
			&h.DocumentTitle, &h.SectionNumber, &h.Content, &h.LegalStatus,
			&h.AccessLevel, &h.Rank); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

var termSanitizer = regexp.MustCompile(`[^\p{L}\p{N}_-]+`)

// ---------- feedback / saved instructions / action routes ----------

func (p *Postgres) AddFeedback(ctx context.Context, answerID, userID, rating, comment string) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO answer_feedback (id, answer_id, user_id, rating, comment)
		VALUES ($1,$2,$3,$4,$5)`, newID("fb"), answerID, nilIfEmpty(userID), rating, comment)
	return err
}

func (p *Postgres) CreateSavedInstruction(ctx context.Context, si model.SavedInstruction) (model.SavedInstruction, error) {
	si.ID = newID("saved")
	si.CreatedAt = time.Now()
	_, err := p.pool.Exec(ctx, `INSERT INTO saved_instructions (id, user_id, answer_id, title, created_at)
		VALUES ($1,$2,$3,$4,$5)`, si.ID, si.UserID, si.AnswerID, si.Title, si.CreatedAt)
	return si, err
}

func (p *Postgres) DeleteSavedInstruction(ctx context.Context, id, userID string) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM saved_instructions WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) ListSavedInstructions(ctx context.Context, userID string) ([]model.SavedInstruction, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, user_id, answer_id, title, created_at
		FROM saved_instructions WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SavedInstruction
	for rows.Next() {
		var s model.SavedInstruction
		if err := rows.Scan(&s.ID, &s.UserID, &s.AnswerID, &s.Title, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (p *Postgres) CreateActionRoute(ctx context.Context, route model.ActionRoute, steps []model.ActionRouteStep) (model.ActionRoute, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.ActionRoute{}, err
	}
	defer tx.Rollback(ctx)

	route.ID = newID("route")
	now := time.Now()
	route.CreatedAt, route.UpdatedAt = now, now
	_, err = tx.Exec(ctx, `INSERT INTO action_routes (id, user_id, answer_id, title, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$5)`, route.ID, route.UserID, route.AnswerID, route.Title, now)
	if err != nil {
		return model.ActionRoute{}, err
	}
	for i := range steps {
		steps[i].ID = newID("rstep")
		steps[i].RouteID = route.ID
		if _, err := tx.Exec(ctx, `INSERT INTO action_route_steps
			(id, route_id, step_id, step_order, title) VALUES ($1,$2,$3,$4,$5)`,
			steps[i].ID, route.ID, nilIfEmpty(steps[i].StepID), steps[i].Order, steps[i].Title); err != nil {
			return model.ActionRoute{}, err
		}
	}
	return route, tx.Commit(ctx)
}

func (p *Postgres) ListActionRoutes(ctx context.Context, userID string) ([]model.ActionRoute, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, user_id, answer_id, title, stale, created_at, updated_at
		FROM action_routes WHERE user_id=$1 ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ActionRoute
	for rows.Next() {
		var r model.ActionRoute
		if err := rows.Scan(&r.ID, &r.UserID, &r.AnswerID, &r.Title, &r.Stale, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *Postgres) GetActionRoute(ctx context.Context, id string) (model.ActionRoute, []model.ActionRouteStep, error) {
	var route model.ActionRoute
	err := p.pool.QueryRow(ctx, `SELECT id, user_id, answer_id, title, stale, created_at, updated_at
		FROM action_routes WHERE id=$1`, id).Scan(
		&route.ID, &route.UserID, &route.AnswerID, &route.Title, &route.Stale, &route.CreatedAt, &route.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ActionRoute{}, nil, ErrNotFound
	}
	if err != nil {
		return model.ActionRoute{}, nil, err
	}
	rows, err := p.pool.Query(ctx, `SELECT id, route_id, COALESCE(step_id,''), step_order, title, done, done_at
		FROM action_route_steps WHERE route_id=$1 ORDER BY step_order`, id)
	if err != nil {
		return route, nil, err
	}
	defer rows.Close()
	var steps []model.ActionRouteStep
	for rows.Next() {
		var s model.ActionRouteStep
		if err := rows.Scan(&s.ID, &s.RouteID, &s.StepID, &s.Order, &s.Title, &s.Done, &s.DoneAt); err != nil {
			return route, nil, err
		}
		steps = append(steps, s)
	}
	return route, steps, rows.Err()
}

func (p *Postgres) SetRouteStepDone(ctx context.Context, routeID, stepID string, done bool) error {
	res, err := p.pool.Exec(ctx, `UPDATE action_route_steps
		SET done=$3, done_at=CASE WHEN $3 THEN now() ELSE NULL END
		WHERE id=$2 AND route_id=$1`, routeID, stepID, done)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	p.pool.Exec(ctx, `UPDATE action_routes SET updated_at=now() WHERE id=$1`, routeID)
	return nil
}

func (p *Postgres) FindDocumentByNumber(ctx context.Context, number string) (model.Document, error) {
	d, err := scanDocument(p.pool.QueryRow(ctx,
		documentProjection+` WHERE d.document_number = $1`, number))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Document{}, ErrNotFound
	}
	return d, err
}

// ---------- audit ----------

func (p *Postgres) AddAuditLog(ctx context.Context, userID, entity, entityID, action, reason string, oldVal, newVal any) error {
	var oldJ, newJ []byte
	var err error
	if oldVal != nil {
		if oldJ, err = json.Marshal(oldVal); err != nil {
			return err
		}
	}
	if newVal != nil {
		if newJ, err = json.Marshal(newVal); err != nil {
			return err
		}
	}
	_, err = p.pool.Exec(ctx, `INSERT INTO audit_logs
		(user_id, entity, entity_id, action, old_value, new_value, reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		nilIfEmpty(userID), entity, entityID, action, oldJ, newJ, reason)
	return err
}
