// Package catalogs expõe as tabelas de catálogo fixas (ano, dificuldade,
// status de questão) usadas para popular seletores no frontend e para
// validar referências no backend. São somente leitura — o conteúdo vem do
// seed da migration (seção 32).
package catalogs

type GradeYear struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Difficulty struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type QuestionStatus struct {
	ID              int64  `json:"id"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	EligibleForExam bool   `json:"eligibleForExam"`
}
