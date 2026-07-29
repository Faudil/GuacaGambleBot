FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /build/bot ./cmd/bot

FROM alpine:3.21

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata su-exec
RUN addgroup -S bot && adduser -S bot -G bot

COPY --from=builder /build/bot ./bot
COPY --from=builder /build/locales ./locales

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD pgrep -f "./bot" || exit 1

ENTRYPOINT ["/bin/sh", "-c", "chown -R bot:bot /app/data 2>/dev/null || true && exec su-exec bot ./bot"]
