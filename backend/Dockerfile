FROM golang:1-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.21
# poppler: текстовый слой PDF; tesseract: OCR сканов (рус+англ)
RUN apk add --no-cache poppler-utils tesseract-ocr tesseract-ocr-data-rus tesseract-ocr-data-eng ca-certificates \
 && adduser -D appuser && mkdir -p /storage && chown appuser /storage
USER appuser
COPY --from=build /bin/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
