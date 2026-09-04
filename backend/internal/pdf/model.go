// Package pdf implementa a geração de provas em PDF (seções 26-30): seleciona
// as questões elegíveis de um caderno, congela um snapshot delas, monta o
// HTML final e aciona o Chromium headless para imprimir em PDF — tudo em
// background, sem bloquear a requisição HTTP que disparou a geração.
package pdf

import (
	"encoding/json"
	"time"
)

// SnapshotAlternative é uma alternativa dentro da representação congelada de
// uma questão (booklet_question_snapshots.alternatives_json) — preservada
// mesmo que a questão original seja editada depois (seção 26).
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

// Kind de generated_documents: cada tipo de prova gera um par de
// documentos — a prova em si e o gabarito daquele tipo, cada um com seu
// próprio arquivo PDF.
const (
	KindExam      = "EXAM"
	KindAnswerKey = "ANSWER_KEY"
)

// Variant é um "tipo de prova" (requisito acrescentado à especificação):
// até 4 por caderno, mesmas questões em todas, só a ordem (por disciplina)
// e a ordem das alternativas mudam entre elas.
type Variant struct {
	ID            int64
	BookletID     int64
	VariantNumber int
}

// VariantQuestionDetail é uma questão já resolvida para UM tipo específico:
// posição impressa naquele tipo, alternativas já na ordem de exibição
// daquele tipo (com IsCorrect recalculado) e a letra correta — que é
// exatamente o gabarito daquela questão naquele tipo.
type VariantQuestionDetail struct {
	SnapshotID        int64
	PositionInVariant int
	DisciplineName    string
	SubjectName       string
	GradeYearName     string
	DifficultyName    string
	StatementJSON     json.RawMessage
	CommandJSON       json.RawMessage
	Alternatives      []SnapshotAlternative
	CorrectLetter     string
}

// GeneratedDocument é um registro de geração de PDF (seção 30) — um por
// tentativa; um caderno pode ter várias (ex.: uma tentativa falhou e foi
// gerada de novo). Desde a introdução dos tipos de prova, cada documento
// pertence a uma variante específica (VariantID) e tem um Kind (prova ou
// gabarito) — um caderno com N tipos gera 2×N documentos por geração.
type GeneratedDocument struct {
	ID            int64
	BookletID     int64
	VariantID     *int64
	VariantNumber *int
	Kind          string
	Status        string
	FilePath      *string
	ErrorMessage  *string
	RequestedBy   int64
	CreatedAt     time.Time
	CompletedAt   *time.Time
}
