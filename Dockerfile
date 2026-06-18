FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

# Migrations are read at runtime by the API for golang-migrate
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /bin/api /usr/local/bin/api

USER app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]