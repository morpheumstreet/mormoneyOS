# syntax=docker/dockerfile:1.7
# mormoneyOS (MoneyClaw) — multi-stage Docker build

# ── Stage 0: Frontend (dashos) ─────────────────────────────────────
FROM oven/bun:1.2-alpine AS frontend
WORKDIR /app
COPY dashos/package.json dashos/bun.lock ./
RUN bun install --frozen-lockfile
COPY dashos/ ./
RUN bun run build:embed

# ── Stage 1: Build ────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build deps (git for module resolution)
RUN apk add --no-cache git ca-certificates

# Copy go mod first for layer caching
COPY go.mod go.sum ./

# Remove local replace for standards (used for dev; use published module in Docker)
RUN go mod edit -dropreplace=github.com/morpheum-labs/standards 2>/dev/null || true

# Copy source
COPY cmd/ cmd/
COPY internal/ internal/

# Copy dashos static assets into embed path (contents of dist into static)
COPY --from=frontend /app/dist/. internal/web/static/

# Build
ARG VERSION
ARG BUILD_TIME
ARG COMMIT
RUN CGO_ENABLED=0 go build -ldflags "-s -w \
  -X github.com/morpheumlabs/mormoneyos-go/cmd.version=${VERSION:-dev} \
  -X github.com/morpheumlabs/mormoneyos-go/cmd.buildTime=${BUILD_TIME:-} \
  -X github.com/morpheumlabs/mormoneyos-go/cmd.commit=${COMMIT:-}" \
  -o /moneyclaw ./cmd/moneyclaw

# ── Stage 2: Release (minimal runtime) ─────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /moneyclaw /usr/local/bin/moneyclaw

# Default data dir (override with AUTOMATON_DIR)
ENV AUTOMATON_DIR=/data
WORKDIR /data

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/moneyclaw"]
CMD ["run"]
