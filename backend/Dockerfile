FROM golang:1-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server && CGO_ENABLED=0 go build -o /bin/worker ./cmd/worker

FROM alpine:3.21
# poppler: текстовый слой PDF; tesseract: OCR сканов (рус+англ)
RUN apk add --no-cache poppler-utils tesseract-ocr tesseract-ocr-data-rus tesseract-ocr-data-eng ca-certificates \
 && adduser -D appuser && mkdir -p /data /storage && chown appuser /data /storage
USER appuser
COPY --from=build /bin/server /server
COPY --from=build /bin/worker /worker
EXPOSE 8080
ENTRYPOINT ["/server"]
