// Package booklets implementa os cadernos de prova de uma aplicação
// (seção 21.1): cada caderno tem configuração, congelamento e numeração
// próprios. Também guarda a configuração padrão (seção 22), copiada para
// a configuração de cada caderno novo.
package booklets

type Booklet struct {
	ID            int64
	ApplicationID int64
	Name          string
	SortOrder     int16
}

// QuotaRule é uma linha de cota final (seção 23): quantidade de questões de
// uma disciplina, opcionalmente refinada por assunto e/ou dificuldade. Cada
// linha é tratada como independente/"folha" — o total de uma disciplina é a
// soma das linhas dela, nunca uma linha-resumo por cima de outras mais
// específicas (decisão registrada no README para não haver ambiguidade na
// validação soma == total_questions da seção 23).
type QuotaRule struct {
	ID           int64
	DisciplineID int64
	SubjectID    *int64
	DifficultyID *int64
	Quantity     int
}

type Configuration struct {
	ID             int64
	BookletID      int64
	TotalQuestions int
	// VariantCount é o número de "tipos de prova" do caderno (1 a 4,
	// padrão 2 — o mais comum): mesmas questões em todos os tipos, só a
	// ordem (por disciplina) e a ordem das alternativas mudam (pacote pdf).
	VariantCount int
	IsFrozen     bool
	GradeYearIDs []int64
	QuotaRules   []QuotaRule
}

// AvailabilityItem é o resultado da validação da seção 24 para uma linha de
// cota: quantas questões elegíveis existem de fato para aquele critério.
type AvailabilityItem struct {
	DisciplineName string
	SubjectName    string
	DifficultyName string
	Requested      int
	Available      int
}

func (a AvailabilityItem) Sufficient() bool {
	return a.Available >= a.Requested
}
