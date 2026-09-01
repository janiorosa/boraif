package questions

import (
	"encoding/json"
	"time"
)

// Question é a entidade estrutural definida na seção 6: enunciado e comando
// são conteúdos independentes (JSON ProseMirror/TipTap — seção 8); as
// alternativas são uma entidade à parte (Alternative).
type Question struct {
	ID             int64
	DisciplineID   int64
	SubjectID      int64
	GradeYearID    int64
	DifficultyID   int64
	StatusID       int64
	AuthorID       int64
	StatementJSON  json.RawMessage
	CommandJSON    json.RawMessage
	RevisionNumber int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Alternative representa uma das cinco posições fixas (1=A .. 5=E — seção 7).
// Não existe uma sexta entidade "correta": IsCorrect é uma propriedade da
// própria alternativa.
type Alternative struct {
	Position    int16
	ContentJSON json.RawMessage
	IsCorrect   bool
}
