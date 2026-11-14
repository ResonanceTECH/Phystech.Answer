#!/usr/bin/env bash
# parse_corpus.sh — единый пайплайн извлечения текста из нормативной базы МФТИ.
#
# PDF:  pdftotext -layout; если текстового слоя нет (< MIN_CHARS символов) →
#       pdftoppm (300 dpi) + tesseract rus+eng. Метки страниц сохраняются
#       как «=== Страница N ===» для привязки цитат.
# DOCX/XLSX: обрабатываются Go-парсером backend'а при инжесте (здесь не
#       дублируем), но для унификации извлекаем .txt через parse_docx.py.
#
# Использование: ./parse_corpus.sh [only-ocr|only-text]
set -euo pipefail
cd "$(dirname "$0")/.."

SRC=data/documents
OUT=data/parsed
PAGES=data/ocr_pages
MIN_CHARS=500          # ниже — считаем PDF сканом
DPI=300
MODE="${1:-all}"

mkdir -p "$OUT" "$PAGES"

pdf_with_text() {
    local f="$1" base="$2"
    pdftotext -layout "$f" "$OUT/$base.txt" 2>/dev/null || true
    local chars
    chars=$(wc -c < "$OUT/$base.txt" | tr -d ' ')
    echo "$chars"
}

pdf_with_ocr() {
    local f="$1" base="$2"
    local dir="$PAGES/$base"
    mkdir -p "$dir"
    echo "  OCR: рендер страниц ($DPI dpi)…"
    pdftoppm -r "$DPI" -png "$f" "$dir/page" 2>/dev/null
    : > "$OUT/$base.txt"
    local i=0
    for img in "$dir"/page-*.png; do
        i=$((i+1))
        printf '=== Страница %d ===\n' "$i" >> "$OUT/$base.txt"
        tesseract "$img" stdout -l rus+eng --psm 6 2>/dev/null >> "$OUT/$base.txt" || true
        printf '\n' >> "$OUT/$base.txt"
    done
    wc -c < "$OUT/$base.txt" | tr -d ' '
}

for f in "$SRC"/*.pdf; do
    base=$(basename "$f" .pdf)
    [ -s "$OUT/$base.txt" ] && { echo "skip $base (уже есть)"; continue; }
    echo "== $base"
    chars=0
    if [ "$MODE" != "only-ocr" ]; then
        chars=$(pdf_with_text "$f" "$base")
    fi
    if [ "$chars" -lt "$MIN_CHARS" ]; then
        chars=$(pdf_with_ocr "$f" "$base")
        echo "  → OCR: $chars символов"
    else
        echo "  → текстовый слой: $chars символов"
    fi
done
echo "Готово. Файлы: $OUT"
