FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /build/bot ./cmd/bot

FROM alpine:3.21

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/bot ./bot
COPY --from=builder /build/locales ./locales

RUN mkdir ./data

CMD ["./bot"]
