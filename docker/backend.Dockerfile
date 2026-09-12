# ── Build stage ──────────────────────────────────────────────────────────
FROM golang:alpine AS builder

WORKDIR /build

# Cache dependency downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /backend ./cmd/backend

# ── Runtime stage ────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /backend .

EXPOSE 9000

ENTRYPOINT ["./backend"]
