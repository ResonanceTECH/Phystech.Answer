FROM golang:1-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/ml ./cmd/ml

FROM alpine:3.21
RUN adduser -D appuser && mkdir -p /data && chown appuser /data
USER appuser
COPY --from=build /bin/ml /ml
# Промпт-шаблоны читаются с диска в рантайме (LLM_PROMPT_DIR).
COPY --from=build /app/prompts /app/prompts
WORKDIR /app
ENV LLM_PROMPT_DIR=/app/prompts
EXPOSE 8000
ENTRYPOINT ["/ml"]
