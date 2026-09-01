// Package pdf implementa a geração de provas em PDF (seções 26-30): seleciona
// as questões elegíveis de um caderno, congela um snapshot delas, monta o
// HTML final e aciona o Chromium headless para imprimir em PDF — tudo em
// background, sem bloquear a requisição HTTP que disparou a geração.
package pdf

import (
	"encoding/json"
	"time"
)

// Snapshot é a representação congelada de uma questão dentro de um caderno
// (seção 26) — preservada mesmo que a questão original seja editada depois.
type Snapshot struct {
	ID                int64
	BookletID         int64
	QuestionID        *int64
	PositionInBooklet int
	DisciplineName    string
	SubjectName       string
	GradeYearName     string
	DifficultyName    string
	StatementJSON     json.RawMessage
	CommandJSON       json.RawMessage
	Alternatives      []SnapshotAlternative
}

type SnapshotAlternative struct {
	Position  string          `json:"position"`
	Content   json.RawMessage `json:"content"`
	IsCorrect bool            `json:"isCorrect"`
}

const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"
)

// GeneratedDocument é um registro de geração de PDF (seção 30) — um por
// tentativa; um caderno pode ter várias (ex.: uma tentativa falhou e foi
// gerada de novo).
type GeneratedDocument struct {
	ID           int64
	BookletID    int64
	Status       string
	FilePath     *string
	ErrorMessage *string
	RequestedBy  int64
	CreatedAt    time.Time
	CompletedAt  *time.Time
}
