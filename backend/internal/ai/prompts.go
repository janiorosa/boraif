package ai

import (
	"fmt"
	"strings"
)

const jsonInstruction = `Você é um assistente pedagógico que revisa questões de prova do Ensino Médio brasileiro.
Você NUNCA reescreve nem substitui o conteúdo do professor — só aponta problemas e sugestões; quem decide aceitar ou não é sempre o professor.
Responda SEMPRE em português do Brasil, em JSON válido, com exatamente estas chaves: {"summary": string, "issues": string[], "suggestions": string[]}. Não inclua nenhum texto fora do JSON.`

// systemPromptFor traduz os critérios específicos de cada alvo (seção 16)
// em instruções para o modelo. O alvo só muda o FOCO da análise; o contexto
// completo da questão sempre é enviado (ver buildUserPrompt), porque avaliar
// o comando, por exemplo, sem saber o enunciado, produz uma crítica pior.
func systemPromptFor(target string) string {
	switch target {
	case "statement":
		return jsonInstruction + "\n\nFoque no ENUNCIADO: clareza, precisão, ambiguidade, gramática, adequação ao ano escolar informado, adequação pedagógica e possíveis problemas factuais."
	case "command":
		return jsonInstruction + "\n\nFoque no COMANDO (a pergunta feita ao aluno): clareza, coerência, objetividade, e se ele realmente pergunta aquilo que as alternativas respondem."
	case "alternatives":
		return jsonInstruction + "\n\nFoque nas ALTERNATIVAS: plausibilidade de cada distrator, existência de pistas óbvias que entregam a resposta, ambiguidade, possibilidade de mais de uma alternativa estar correta, e qualidade da alternativa marcada como correta."
	case "full":
		return jsonInstruction + "\n\nFoque na QUESTÃO INTEIRA: coerência geral, relação entre enunciado/comando/alternativas, se existe exatamente uma alternativa correta e se ela é a mais adequada, adequação ao ano e à dificuldade informados, qualidade pedagógica e clareza geral."
	default:
		return jsonInstruction
	}
}

func buildUserPrompt(gradeYear, difficulty, statement, command string, alternatives []AlternativeInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Ano do Ensino Médio: %s\nDificuldade pretendida: %s\n\n", orDash(gradeYear), orDash(difficulty))

	if strings.TrimSpace(statement) != "" {
		fmt.Fprintf(&b, "Enunciado:\n%s\n\n", statement)
	}
	if strings.TrimSpace(command) != "" {
		fmt.Fprintf(&b, "Comando:\n%s\n\n", command)
	}
	if len(alternatives) > 0 {
		b.WriteString("Alternativas:\n")
		for _, a := range alternatives {
			marker := ""
			if a.IsCorrect {
				marker = "  <- marcada como correta"
			}
			fmt.Fprintf(&b, "%s) %s%s\n", a.Position, a.Text, marker)
		}
	}
	return b.String()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(não informado)"
	}
	return s
}
