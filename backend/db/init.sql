-- Инициализация БД Физтех.Ответ. Выполняется под суперпользователем.
-- Запуск:
--   psql -h localhost -U postgres -d postgres \
--        -v pw='<пароль fiztech_admin>' -f backend/db/init.sql
-- Пароль передаётся параметром :pw и в файле не хранится.

\set ON_ERROR_STOP on

-- Роль приложения: только LOGIN, без SUPERUSER/CREATEDB/CREATEROLE.
SELECT 'CREATE ROLE fiztech_admin LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiztech_admin')\gexec

ALTER ROLE fiztech_admin LOGIN PASSWORD :'pw';

-- База, владелец — fiztech_admin. Существующая база не трогается.
SELECT 'CREATE DATABASE fiztech_answer OWNER fiztech_admin ENCODING ''UTF8'' TEMPLATE template0'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'fiztech_answer')\gexec

GRANT ALL PRIVILEGES ON DATABASE fiztech_answer TO fiztech_admin;

\connect fiztech_answer

GRANT CREATE ON SCHEMA public TO fiztech_admin;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- Раскомментировать после установки pgvector:
-- CREATE EXTENSION IF NOT EXISTS vector;
