-- Авторизация и профили (спека §12): поля профиля пользователя, сессии по
-- хешу токена, одноразовые токены сброса пароля, справочник подразделений.
-- Идемпотентна.

-- ---------- users: профиль ----------

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS first_name         text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_name          text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS employee_category  text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS department_id      text,
    ADD COLUMN IF NOT EXISTS "position"         text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS onboarding_complete boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS email_verified_at  timestamptz,
    ADD COLUMN IF NOT EXISTS updated_at         timestamptz NOT NULL DEFAULT now();

-- Уже существующие аккаунты считаем настроенными; имя разбиваем на части.
UPDATE users
SET first_name = split_part(name, ' ', 1),
    last_name  = coalesce(nullif(btrim(substr(name, length(split_part(name, ' ', 1)) + 1)), ''), ''),
    onboarding_complete = true
WHERE name <> '' AND first_name = '';

-- Нормализованная почта уникальна (регистрация идёт по email).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower
    ON users(lower(email)) WHERE email <> '';

-- login остаётся для ранних аккаунтов; у новых пользователей он пустой.
-- '' под UNIQUE даёт ложный конфликт — храним NULL и уникальность
-- оставляем только для заполненных логинов.
ALTER TABLE users ALTER COLUMN login DROP NOT NULL;
UPDATE users SET login = NULL WHERE login = '';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_login_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_login_nonempty
    ON users(login) WHERE login IS NOT NULL;

-- ---------- sessions: хеш токена, активность, отзыв ----------

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS token_hash   text,
    ADD COLUMN IF NOT EXISTS last_seen_at timestamptz,
    ADD COLUMN IF NOT EXISTS revoked_at   timestamptz;

-- В БД хранится только хеш непрозрачного идентификатора сессии.
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash
    ON sessions(token_hash) WHERE token_hash IS NOT NULL;

-- ---------- password_reset_tokens ----------

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_prt_hash ON password_reset_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_prt_user ON password_reset_tokens(user_id);

-- ---------- departments ----------

CREATE TABLE IF NOT EXISTS departments (
    id         text PRIMARY KEY,
    name       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_department'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT fk_users_department
            FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Справочник подразделений МФТИ.
INSERT INTO departments (id, name) VALUES
    ('dep_falt', 'ФАЛТ — физтех-школа аэро-космических технологий'),
    ('dep_fpmi', 'ФПМИ — физтех-школа прикладной математики и информатики'),
    ('dep_frkt', 'ФРКТ — физтех-школа радиотехники и компьютерных технологий'),
    ('dep_fefm', 'ФЭФМ — физтех-школа электроники, фотоники и молекулярной физики'),
    ('dep_fbmf', 'ФБМФ — физтех-школа биологической и медицинской физики'),
    ('dep_lfi',  'ЛФИ — лаборатория физики высоких энергий'),
    ('dep_faki', 'ФАКИ — факультет общей и прикладной физики'),
    ('dep_fmhf', 'ФМХФ — физтех-школа материаловедения и химической физики'),
    ('dep_vshm', 'ВШМ — высшая школа математики'),
    ('dep_vshpi','ВШПИ — высшая школа программной инженерии'),
    ('dep_cit',  'ЦИТ — центр информационных технологий'),
    ('dep_uo',   'Учебный отдел'),
    ('dep_hr',   'Отдел кадров'),
    ('dep_acc',  'Бухгалтерия'),
    ('dep_campus','Дирекция кампуса')
ON CONFLICT (id) DO NOTHING;
