module boraif

go 1.25

require (
	// v5.9.0+ obrigatório: versões anteriores têm CVE-2026-33816 (crítico).
	github.com/jackc/pgx/v5 v5.9.2
	github.com/pressly/goose/v3 v3.7.0
	golang.org/x/crypto v0.28.0
	// Controla o Chromium headless para geração de PDF (seção 28/29).
	// cdproto não é listado aqui de propósito: "go mod tidy" (rodado no
	// Dockerfile) resolve a versão certa sozinho a partir do import direto
	// em internal/pdf, sem eu precisar adivinhar um pseudo-version exato.
	github.com/chromedp/chromedp v0.11.0
)
