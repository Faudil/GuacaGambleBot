FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Guarantee the assets directory exists even in checkouts without it (e.g.
# CI before the images are committed), so stage-1's COPY never breaks.
RUN mkdir -p /build/assets
RUN go build -o /build/bot ./cmd/bot

FROM alpine:3.21

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata su-exec
RUN addgroup -S bot && adduser -S bot -G bot

COPY --from=builder /build/bot ./bot
COPY --from=builder /build/locales ./locales
COPY --from=builder /build/assets ./assets

# The bot's watchdog touches /tmp/bot.heartbeat on every healthy check. A stale
# heartbeat first reports unhealthy (passive), then escalates to a full reboot
# (SIGQUIT on PID 1 -> dump + exit -> restart: always) when the process itself
# is wedged and can no longer run its watchdog.
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
  CMD sh -c 'f=/tmp/bot.heartbeat; [ ! -f "$f" ] && exit 1; \
    age=$(($(date +%s) - $(stat -c %Y "$f"))); \
    [ "$age" -lt 45 ] && exit 0; \
    [ "$age" -gt 90 ] && kill -QUIT 1; \
    exit 1'

ENTRYPOINT ["/bin/sh", "-c", "chown -R bot:bot /app/data /app/assets 2>/dev/null || true && exec su-exec bot ./bot"]
