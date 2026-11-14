-- Трассировка RAG-конвейера («Как найден ответ»): режим поиска, подвопросы,
-- изменяющие акты, модель, время этапов. JSON как есть из ML-сервиса.
ALTER TABLE answers ADD COLUMN IF NOT EXISTS trace jsonb;
