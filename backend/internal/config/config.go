// Package config carrega a configuração do backend a partir de variáveis de
// ambiente. Nenhum segredo tem valor padrão embutido no código.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	SessionCookie string
	// CookieSecure deve ser true em produção (HTTPS). Em desenvolvimento local
	// sem TLS, pode ser desativado via COOKIE_SECURE=false.
	CookieSecure bool
	// UploadsDir é a raiz onde imagens (e, futuramente, PDFs) ficam no
	// filesystem (seção 31), servida publicamente em /uploads/.
	UploadsDir string
	// MaxUploadSizeBytes limita o tamanho de cada imagem enviada (seção 35).
	MaxUploadSizeBytes int64
	// APIKeyEncryptionKey cifra as API Keys da OpenAI dos professores
	// (seção 17). Exatamente 32 bytes, nunca guardada no Postgres.
	APIKeyEncryptionKey []byte
	// OpenAIModel é o modelo usado pelo assistente de revisão (seção 16).
	OpenAIModel string
	// GeneratedDir é a raiz onde os PDFs gerados ficam no filesystem
	// (seção 31), dentro de generated/applications/{app}/{caderno}/.
	GeneratedDir string
	// ChromePath é o binário do Chromium usado para gerar PDF (seção 29).
	ChromePath string
}

func Load() (Config, error) {
	maxUploadMB, err := strconv.Atoi(getEnv("MAX_UPLOAD_SIZE_MB", "5"))
	if err != nil || maxUploadMB <= 0 {
		maxUploadMB = 5
	}

	cfg := Config{
		Port:               getEnv("BACKEND_PORT", "8080"),
		SessionCookie:      getEnv("SESSION_COOKIE_NAME", "boraif_session"),
		CookieSecure:       getEnv("COOKIE_SECURE", "true") != "false",
		UploadsDir:         getEnv("UPLOADS_DIR", "uploads"),
		MaxUploadSizeBytes: int64(maxUploadMB) << 20,
		GeneratedDir:       getEnv("GENERATED_DIR", "generated"),
		ChromePath:         getEnv("CHROME_PATH", "/usr/bin/chromium"),
	}

	dbURL, err := requireEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseURL = dbURL

	encSecret, err := requireEnv("API_KEY_ENCRYPTION_SECRET")
	if err != nil {
		return Config{}, err
	}
	encKey, err := base64.StdEncoding.DecodeString(encSecret)
	if err != nil {
		return Config{}, fmt.Errorf("API_KEY_ENCRYPTION_SECRET must be valid base64: %w", err)
	}
	if len(encKey) != 32 {
		return Config{}, fmt.Errorf("API_KEY_ENCRYPTION_SECRET must decode to exactly 32 bytes (AES-256), got %d", len(encKey))
	}
	cfg.APIKeyEncryptionKey = encKey
	cfg.OpenAIModel = getEnv("OPENAI_MODEL", "gpt-4o-mini")

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("missing required environment variable %s", key)
	}
	return v, nil
}
