# ── Stage 1: Build the Go runner binary ──────────────────────────────────────
FROM golang:1.23-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
      -ldflags "-s -w -X main.Version=$(cat VERSION)" \
      -o /shellrelay ./cmd/shellrelay

# ── Stage 2: Ubuntu 24.04 runtime ───────────────────────────────────────────
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
      bash \
      ca-certificates \
      curl \
      htop \
      vim-tiny \
      net-tools \
      iputils-ping \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /shellrelay /usr/local/bin/shellrelay
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create a non-root user for the terminal sessions
RUN useradd -m -s /bin/bash shellrelay

USER shellrelay
WORKDIR /home/shellrelay

# Config via env vars (set in docker-compose or docker run)
ENV SHELLRELAY_URL=wss://prod-api.shellrelay.com
ENV SHELLRELAY_SERVER_ID=""
ENV SHELLRELAY_TOKEN=""
ENV SHELLRELAY_EMAIL=""
ENV SHELLRELAY_SERVER_NAME=""

# Entrypoint handles both manual mode (TOKEN set) and announce mode (EMAIL set)
ENTRYPOINT ["entrypoint.sh"]
