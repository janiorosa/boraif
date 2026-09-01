package images

import "time"

// Image é uma imagem enviada por um professor, compartilhada com todos os
// professores da mesma disciplina (seção 13) — não há autoria individual
// nem permissão complexa: a separação é só por disciplina.
type Image struct {
	ID           int64
	DisciplineID int64
	Filename     string
	Path         string
	MimeType     string
	SizeBytes    int64
	UploadedBy   int64
	CreatedAt    time.Time
}
