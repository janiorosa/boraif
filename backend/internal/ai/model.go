// Package ai implementa o assistente de revisão por IA (seção 16). A IA
// NUNCA gera nem substitui conteúdo — só analisa o que o professor já
// escreveu e devolve críticas/sugestões; a decisão de aceitar ou não é
// sempre do professor.
package ai

// ReviewResult é a resposta estruturada da IA. Mantida deliberadamente
// simples (resumo + listas) em vez de um campo rígido por critério da
// seção 16 — os critérios entram no prompt (ver prompts.go), guiando a
// análise, sem forçar a IA a preencher campos que não se aplicam.
type ReviewResult struct {
	Summary     string   `json:"summary"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

// AlternativeInput é o texto simples de uma alternativa (já extraído do
// ProseMirror pelo frontend via editor.getText()) mais sua posição e se
// está marcada como correta — contexto necessário para avaliar distratores.
type AlternativeInput struct {
	Position  string
	Text      string
	IsCorrect bool
}
