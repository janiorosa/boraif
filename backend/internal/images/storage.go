package images

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage grava arquivos no filesystem local (seção 31 — solução inicial,
// sem nuvem). É deliberadamente simples para permitir uma futura migração
// para S3 ou equivalente, sem implementar isso agora.
type Storage struct {
	// BaseDir é a raiz de uploads (ex.: "uploads"), a mesma raiz servida
	// publicamente em /uploads/.
	BaseDir string
}

func NewStorage(baseDir string) *Storage {
	return &Storage{BaseDir: baseDir}
}

// Save grava o conteúdo em {BaseDir}/images/{disciplineCode}/{nome-aleatório}{ext}
// e devolve o caminho relativo a BaseDir (guardado no banco e usado para
// montar a URL pública /uploads/{relativePath}).
//
// O nome do arquivo nunca vem do cliente: é gerado aqui a partir de bytes
// aleatórios, o que evita path traversal e nomes perigosos (seção 35).
func (s *Storage) Save(disciplineCode, ext string, r io.Reader) (relativePath string, err error) {
	subdir := filepath.Join("images", disciplineCode)
	dir := filepath.Join(s.BaseDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating upload directory: %w", err)
	}

	name, err := randomFilename(ext)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(dir, name)
	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	return filepath.ToSlash(filepath.Join(subdir, name)), nil
}

func randomFilename(ext string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw) + ext, nil
}
