FROM golang:1-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.21
RUN adduser -D appuser && mkdir -p /data && chown appuser /data
USER appuser
COPY --from=build /bin/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
