#!/usr/bin/env bash
# download_models.sh — скачивает GGUF-веса в ml/models/.
#
# Веса не хранятся в git (см. ml/.gitignore, ~6 ГБ суммарно): ml-сервис
# сам их не читает — он ходит по HTTP на OpenAI-compatible endpoints
# llama.cpp/Ollama. Скрипт просто раскладывает файлы для локального запуска.
#
# Использование:
#   ml/scripts/download_models.sh              # весь набор
#   ml/scripts/download_models.sh qwen3.5      # только файлы, где есть подстрока
#
# Уже скачанные файлы пропускаются. При обрыве недокачанный файл остаётся
# как *.part — повторный запуск докачает с места обрыва (curl -C -).

set -euo pipefail
cd "$(dirname "$0")/.."

BASE="https://huggingface.co"

# <путь от ml/models> | <hf-репозиторий> | <файл в репозитории> | <описание>
MODELS=(
  "llm/Qwen3.5-4B-Q4_K_M.gguf|unsloth/Qwen3.5-4B-GGUF|Qwen3.5-4B-Q4_K_M.gguf|LLM: чат/reasoning, llama-server :8081"
  "llm/Qwen3-4B-Instruct-2507-Q4_K_M.gguf|lmstudio-community/Qwen3-4B-Instruct-2507-GGUF|Qwen3-4B-Instruct-2507-Q4_K_M.gguf|LLM: instruct-альтернатива"
  "embedding/Qwen3-Embedding-0.6B-Q8_0.gguf|Qwen/Qwen3-Embedding-0.6B-GGUF|Qwen3-Embedding-0.6B-Q8_0.gguf|эмбеддинги, llama-server :8082 --embedding"
  "reranker/qwen3-reranker-0.6b-q8_0.gguf|ggml-org/Qwen3-Reranker-0.6B-Q8_0-GGUF|qwen3-reranker-0.6b-q8_0.gguf|реранкер, llama-server :8083 --reranking"
)

want=("$@")
failed=0
for entry in "${MODELS[@]}"; do
  IFS='|' read -r rel repo file desc <<< "$entry"
  dest="models/$rel"

  if [ "${#want[@]}" -gt 0 ]; then
    match=0
    for w in "${want[@]}"; do
      [[ "$rel" == *"$w"* || "$file" == *"$w"* ]] && match=1
    done
    [ "$match" -eq 0 ] && continue
  fi

  if [ -f "$dest" ] && [ -s "$dest" ]; then
    printf 'skip  %s (уже скачан)\n' "$dest"
    continue
  fi

  mkdir -p "$(dirname "$dest")"
  url="$BASE/$repo/resolve/main/$file"
  tmp="$dest.part"
  echo ">>> $desc"
  echo "    $url"

  if curl -fL --retry 3 --retry-delay 5 -C - -o "$tmp" "$url"; then
    mv "$tmp" "$dest"
    printf 'ok    %s (%s)\n' "$dest" "$(du -h "$dest" | cut -f1)"
  else
    echo "FAIL  $dest — частичный файл сохранён как $tmp, повторный запуск докачает" >&2
    failed=1
  fi
done

if [ "$failed" -eq 0 ]; then
  echo "Готово: $(du -sh models | cut -f1) в ml/models/"
fi
exit "$failed"
