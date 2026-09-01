# --- build stage ---------------------------------------------------------
# 1.25+ exigido pelo go.mod do pgx/v5 (v5.9.0+, obrigatório por causa da
# CVE-2026-33816) — Go 1.23 falha o build com GOTOOLCHAIN=local.
FROM golang:1.25-bookworm AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /out/server ./cmd/server

# --- runtime stage ---------------------------------------------------------
# Debian slim porque o Chromium usado na geração de PDF (Fase 10) precisa de
# bibliotecas de sistema; evita reconstruir a base de imagem mais tarde.
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    ca-certificates \
    fonts-liberation \
    fonts-noto \
    && rm -rf /var/lib/apt/lists/*

ENV CHROME_PATH=/usr/bin/chromium

WORKDIR /app
COPY --from=build /out/server /app/server

RUN mkdir -p /app/uploads /app/generated

EXPOSE 8080
ENTRYPOINT ["/app/server"]
