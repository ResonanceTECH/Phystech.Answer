-- FTS-индекс по lower(content): русский стеммер не нормализует слова с
-- заглавной буквы ('Аванс' != 'аванс'), поэтому индекс и запросы идут по
-- lower(content).
DROP INDEX IF EXISTS idx_chunks_fts;
CREATE INDEX IF NOT EXISTS idx_chunks_fts ON document_chunks
    USING gin(to_tsvector('russian', lower(content)));
