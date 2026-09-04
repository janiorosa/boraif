package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Storage grava o PDF gerado no filesystem (seção 31), organizado por
// aplicação, caderno e tipo de prova — mesma técnica simples de
// internal/images/storage.go.
type Storage struct {
	BaseDir string // raiz de "generated", ex.: "generated"
}

func NewStorage(baseDir string) *Storage {
	return &Storage{BaseDir: baseDir}
}

// Save grava em
// {BaseDir}/applications/{applicationID}/{bookletID}/tipo-{variantNumber}/{kind}-{documentID}.pdf
// e devolve o caminho relativo a BaseDir (guardado em generated_documents.file_path).
func (s *Storage) Save(applicationID, bookletID int64, variantNumber int, kind string, documentID int64, data []byte) (relativePath string, err error) {
	subdir := filepath.Join("applications", fmt.Sprint(applicationID), fmt.Sprint(bookletID), fmt.Sprintf("tipo-%d", variantNumber))
	dir := filepath.Join(s.BaseDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating generated directory: %w", err)
	}

	name := fmt.Sprintf("%s-%d.pdf", strings.ToLower(kind), documentID)
	fullPath := filepath.Join(dir, name)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("writing pdf file: %w", err)
	}

	return filepath.ToSlash(filepath.Join(subdir, name)), nil
}

// FullPath resolve um caminho relativo (guardado no banco) para um caminho
// absoluto no filesystem, usado para servir o download do PDF.
func (s *Storage) FullPath(relativePath string) string {
	return filepath.Join(s.BaseDir, filepath.FromSlash(relativePath))
}
