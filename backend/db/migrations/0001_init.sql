-- Базовая схема Физтех.Ответ (спека §21). Идемпотентна: IF NOT EXISTS.

CREATE TABLE IF NOT EXISTS users (
    id            text PRIMARY KEY,
    name          text NOT NULL,
    email         text NOT NULL DEFAULT '',
    login         text NOT NULL DEFAULT '' UNIQUE,
    password_hash text NOT NULL DEFAULT '',
    role          text NOT NULL DEFAULT 'employee', -- employee|admin|expert
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS source_files (
    id                text PRIMARY KEY,
    original_filename text NOT NULL,
    storage_key       text NOT NULL,
    mime_type         text NOT NULL DEFAULT '',
    size_bytes        bigint NOT NULL DEFAULT 0,
    sha256            text NOT NULL DEFAULT '',
    uploaded_by       text REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_source_files_sha ON source_files(sha256) WHERE sha256 <> '';

CREATE TABLE IF NOT EXISTS documents (
    id              text PRIMARY KEY,
    title           text NOT NULL,
    document_number text NOT NULL DEFAULT '',
    document_type   text NOT NULL DEFAULT '',  -- приказ|регламент|распоряжение|приложение|форма
    category        text NOT NULL DEFAULT '',
    document_date   date,
    access_level    text NOT NULL DEFAULT 'internal', -- public|internal|confidential|restricted
    tags            text[] NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_documents_number ON documents(document_number);
CREATE INDEX IF NOT EXISTS idx_documents_category ON documents(category);

CREATE TABLE IF NOT EXISTS document_versions (
    id                text PRIMARY KEY,
    document_id       text NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    source_file_id    text REFERENCES source_files(id) ON DELETE SET NULL,
    version_number    int NOT NULL DEFAULT 1,
    content           text NOT NULL DEFAULT '',
    effective_from    date,
    effective_to      date,
    legal_status      text NOT NULL DEFAULT 'in_force',   -- in_force|repealed|superseded|draft
    review_status     text NOT NULL DEFAULT 'approved',   -- pending|approved|requires_review
    processing_status text NOT NULL DEFAULT 'published',  -- pending|processing|published|requires_review|failed
    processing_error  text NOT NULL DEFAULT '',
    published_at      timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_doc_versions_document ON document_versions(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_versions_status ON document_versions(processing_status);

CREATE TABLE IF NOT EXISTS document_sections (
    id             text PRIMARY KEY,
    version_id     text NOT NULL REFERENCES document_versions(id) ON DELETE CASCADE,
    parent_id      text REFERENCES document_sections(id) ON DELETE CASCADE,
    section_number text NOT NULL DEFAULT '',
    title          text NOT NULL DEFAULT '',
    content        text NOT NULL DEFAULT '',
    page_number    int,
    position       int NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_sections_version ON document_sections(version_id);

CREATE TABLE IF NOT EXISTS document_chunks (
    id                text PRIMARY KEY,
    version_id        text NOT NULL REFERENCES document_versions(id) ON DELETE CASCADE,
    section_id        text REFERENCES document_sections(id) ON DELETE SET NULL,
    content           text NOT NULL,
    chunk_index       int NOT NULL DEFAULT 0,
    location_type     text NOT NULL DEFAULT 'text', -- text|page|sheet
    page_number       int,
    sheet_name        text NOT NULL DEFAULT '',
    cell_range        text NOT NULL DEFAULT '',
    location_metadata jsonb NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_chunks_version ON document_chunks(version_id);
CREATE INDEX IF NOT EXISTS idx_chunks_fts ON document_chunks USING gin(to_tsvector('russian', content));

-- Эмбеддинги хранит ML-сервис; таблица предусмотрена схемой для pgvector-позднего.
CREATE TABLE IF NOT EXISTS chunk_embeddings (
    id         text PRIMARY KEY,
    chunk_id   text NOT NULL REFERENCES document_chunks(id) ON DELETE CASCADE,
    model_id   text NOT NULL DEFAULT '',
    embedding  real[],
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS document_relations (
    id                 text PRIMARY KEY,
    source_version_id  text REFERENCES document_versions(id) ON DELETE CASCADE,
    target_document_id text REFERENCES documents(id) ON DELETE CASCADE,
    target_version_id  text REFERENCES document_versions(id) ON DELETE SET NULL,
    relation_type      text NOT NULL, -- approves|amends|repeals|replaces|supplements|references|has_attachment
    evidence_chunk_id  text REFERENCES document_chunks(id) ON DELETE SET NULL,
    effective_from     date,
    review_status      text NOT NULL DEFAULT 'pending', -- pending|confirmed|rejected
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_relations_source ON document_relations(source_version_id);
CREATE INDEX IF NOT EXISTS idx_relations_target ON document_relations(target_document_id);

CREATE TABLE IF NOT EXISTS conversations (
    id         text PRIMARY KEY,
    user_id    text REFERENCES users(id) ON DELETE SET NULL,
    title      text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_conversations_user ON conversations(user_id);

CREATE TABLE IF NOT EXISTS messages (
    id              text PRIMARY KEY,
    conversation_id text NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            text NOT NULL, -- user|assistant|system
    content         text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);

CREATE TABLE IF NOT EXISTS answer_runs (
    id              text PRIMARY KEY,
    conversation_id text NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id      text REFERENCES messages(id) ON DELETE SET NULL,
    status          text NOT NULL DEFAULT 'queued', -- queued|running|completed|failed|cancelled
    error_code      text NOT NULL DEFAULT '',
    started_at      timestamptz,
    completed_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_runs_conversation ON answer_runs(conversation_id);

CREATE TABLE IF NOT EXISTS run_events (
    id         bigserial PRIMARY KEY,
    run_id     text NOT NULL REFERENCES answer_runs(id) ON DELETE CASCADE,
    event_type text NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_run_events_run ON run_events(run_id, id);

CREATE TABLE IF NOT EXISTS answers (
    id                  text PRIMARY KEY,
    run_id              text NOT NULL REFERENCES answer_runs(id) ON DELETE CASCADE,
    status              text NOT NULL, -- answered|partial|clarification_required|not_found|conflict|error
    summary             text NOT NULL DEFAULT '',
    warnings            text[] NOT NULL DEFAULT '{}',
    missing_information text[] NOT NULL DEFAULT '{}',
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS answer_steps (
    id             text PRIMARY KEY,
    answer_id      text NOT NULL REFERENCES answers(id) ON DELETE CASCADE,
    step_order     int NOT NULL DEFAULT 0,
    title          text NOT NULL DEFAULT '',
    description    text NOT NULL DEFAULT '',
    responsible    text,
    deadline       text,
    condition_text text,
    required       boolean NOT NULL DEFAULT true
);
CREATE INDEX IF NOT EXISTS idx_steps_answer ON answer_steps(answer_id);

CREATE TABLE IF NOT EXISTS answer_citations (
    id                  text PRIMARY KEY,
    answer_id           text NOT NULL REFERENCES answers(id) ON DELETE CASCADE,
    step_id             text REFERENCES answer_steps(id) ON DELETE CASCADE,
    chunk_id            text REFERENCES document_chunks(id) ON DELETE SET NULL,
    document_version_id text REFERENCES document_versions(id) ON DELETE SET NULL,
    document_id         text REFERENCES documents(id) ON DELETE SET NULL,
    document_title      text NOT NULL DEFAULT '',
    section_number      text NOT NULL DEFAULT '',
    page_number         int,
    quoted_text         text NOT NULL DEFAULT '',
    score               double precision NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_citations_answer ON answer_citations(answer_id);
CREATE INDEX IF NOT EXISTS idx_citations_chunk ON answer_citations(chunk_id);

CREATE TABLE IF NOT EXISTS processing_jobs (
    id                  text PRIMARY KEY,
    document_version_id text REFERENCES document_versions(id) ON DELETE CASCADE,
    job_type            text NOT NULL, -- document.parse|document.index|answer.generate|...
    status              text NOT NULL DEFAULT 'queued', -- queued|running|succeeded|failed|cancelled
    payload             jsonb NOT NULL DEFAULT '{}',
    progress            int NOT NULL DEFAULT 0,
    attempts            int NOT NULL DEFAULT 0,
    max_attempts        int NOT NULL DEFAULT 3,
    retry_at            timestamptz,
    locked_at           timestamptz,
    error_code          text NOT NULL DEFAULT '',
    idempotency_key     text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    completed_at        timestamptz
);
CREATE INDEX IF NOT EXISTS idx_jobs_status_retry ON processing_jobs(status, retry_at);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON processing_jobs(job_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_idem ON processing_jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_doc_reads (
    user_id     text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_id text NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    read_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, document_id)
);

CREATE TABLE IF NOT EXISTS saved_instructions (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    answer_id  text NOT NULL REFERENCES answers(id) ON DELETE CASCADE,
    title      text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS document_conflicts (
    id                 text PRIMARY KEY,
    conflict_type      text NOT NULL, -- deadline_conflict|responsible_conflict|requirement_conflict|version_unclear|missing_document
    status             text NOT NULL DEFAULT 'unresolved', -- unresolved|resolved|dismissed
    description        text NOT NULL DEFAULT '',
    source_version_ids text[] NOT NULL DEFAULT '{}',
    requires_review    boolean NOT NULL DEFAULT true,
    created_at         timestamptz NOT NULL DEFAULT now(),
    resolved_at        timestamptz
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id         bigserial PRIMARY KEY,
    user_id    text REFERENCES users(id) ON DELETE SET NULL,
    entity     text NOT NULL,
    entity_id  text NOT NULL,
    action     text NOT NULL,
    old_value  jsonb,
    new_value  jsonb,
    reason     text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
