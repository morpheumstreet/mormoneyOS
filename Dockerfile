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
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build deps (git for module resolution)
RUN apk add --no-cache git ca-certificates

# Copy go mod first for layer caching
COPY go.mod go.sum ./

# Remove local replace for standards (used for dev; use published module in Docker)
RUN go mod edit -dropreplace=github.com/morpheum-labs/standards 2>/dev/null || true

# Auth for private GitHub repos (github.com/morpheum-labs/*)
# Pass via: docker build --build-arg GITHUB_TOKEN=$GITHUB_TOKEN
# Safer: docker build --secret id=github_token,env=GITHUB_TOKEN (use RUN --mount=type=secret)
ARG GITHUB_TOKEN
ENV GOPRIVATE=github.com/morpheum-labs
RUN if [ -n "${GITHUB_TOKEN}" ]; then \
      git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; \
    fi

# Fetch modules with auth in place
RUN go mod download

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

# ── Stage 2: Release (minimal runtime with shell) ───────────────────
FROM alpine:3.19

RUN apk add --no-cache bash ca-certificates

COPY --from=builder /moneyclaw /usr/local/bin/moneyclaw

# Non-root user (UID 1000)
RUN adduser -D -u 1000 mormusr

# Default data dir (override with AUTOMATON_DIR)
ENV AUTOMATON_DIR=/data
WORKDIR /data

EXPOSE 8080

# Ensure data dir is writable by mormusr
RUN chown -R mormusr:mormusr /data

USER mormusr

# Allow bash and direct binary commands; default to moneyclaw run
ENTRYPOINT ["/usr/local/bin/moneyclaw"]
CMD ["run"]
