FROM golang:1-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/ml ./cmd/ml && CGO_ENABLED=0 go build -o /bin/eval ./cmd/eval

FROM alpine:3.21
RUN adduser -D appuser && mkdir -p /data && chown appuser /data
WORKDIR /data
USER appuser
COPY --from=build /bin/ml /ml
COPY --from=build /bin/eval /eval
EXPOSE 8000
ENTRYPOINT ["/ml"]
