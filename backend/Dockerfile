FROM golang:1-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.21
RUN adduser -D appuser
USER appuser
COPY --from=build /bin/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
