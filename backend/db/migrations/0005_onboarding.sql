-- Ознакомление с регламентами: блокировка участников, видимость документов,
-- назначения, уведомления. Отделы и профиль (department_id, position) —
-- в 0004_auth_profiles. documents.category используется как тематическая папка.

ALTER TABLE users ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT true;

-- active — виден всем и ассистенту; hidden — временно скрыт (например, до
-- проверки); archived — утратил актуальность, хранится для истории.
ALTER TABLE documents ADD COLUMN IF NOT EXISTS visibility text NOT NULL DEFAULT 'active';
CREATE INDEX IF NOT EXISTS idx_documents_visibility ON documents(visibility);

-- Кому документ назначен для ознакомления. target_type: all | department | user.
-- requirement: required (обязательный, может иметь срок) | optional (рекомендуемый, в KPI).
CREATE TABLE IF NOT EXISTS document_assignments (
    id          text PRIMARY KEY,
    document_id text NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    target_type text NOT NULL,
    target_id   text NOT NULL DEFAULT '',
    requirement text NOT NULL DEFAULT 'required',
    due_date    date,
    assigned_by text REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (document_id, target_type, target_id)
);
CREATE INDEX IF NOT EXISTS idx_assignments_target ON document_assignments(target_type, target_id);

-- kind: assigned | new_document | updated
CREATE TABLE IF NOT EXISTS notifications (
    id          text PRIMARY KEY,
    user_id     text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        text NOT NULL,
    document_id text REFERENCES documents(id) ON DELETE CASCADE,
    title       text NOT NULL,
    body        text NOT NULL DEFAULT '',
    read_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC);
