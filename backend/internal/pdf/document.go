package pdf

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// htmlHead/htmlTail ficam fora de qualquer fmt.Sprintf: o CSS tem "%"
// literal (max-width: 100%), o que quebraria verbos de formatação se o
// template inteiro passasse por Sprintf. Título e corpo são concatenados
// diretamente (strings.Builder), sem esse risco.
const htmlHead = `<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8" />
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.47/dist/katex.min.css">
<script src="https://cdn.jsdelivr.net/npm/katex@0.16.47/dist/katex.min.js"></script>
<style>
  body { font-family: "Helvetica Neue", Arial, sans-serif; font-size: 12pt; color: #111; margin: 0; }
  h1 { font-size: 14pt; }
  .question { margin-bottom: 18pt; page-break-inside: avoid; }
  .question-number { font-weight: bold; margin-bottom: 4pt; }
  .statement, .command { margin-bottom: 8pt; }
  .alternatives { list-style-type: upper-alpha; padding-left: 20pt; }
  .alternatives li { margin-bottom: 4pt; }
  table { border-collapse: collapse; margin: 8pt 0; }
  table td, table th { border: 1px solid #333; padding: 4pt 8pt; }
  img { max-width: 100%; }
  .variant-badge {
    display: inline-block; font-size: 18pt; font-weight: bold;
    border: 1.5pt solid #111; padding: 4pt 12pt; margin-bottom: 10pt;
  }
  .answer-key { column-width: 110pt; column-gap: 24pt; margin-top: 8pt; }
  .answer-item { break-inside: avoid; margin-bottom: 6pt; font-size: 12pt; }
</style>
</head>
<body>
`

// htmlTail roda o KaTeX sobre cada placeholder ".math" (ver render.go) e só
// então marca a página como pronta — o Chromium (chromium.go) espera esse
// marcador antes de imprimir, para nunca gerar um PDF com fórmula sem
// renderizar.
const htmlTail = `
<script>
  document.querySelectorAll(".math").forEach(function (el) {
    try {
      katex.render(el.getAttribute("data-latex") || "", el, {
        throwOnError: false,
        displayMode: el.tagName === "DIV",
      });
    } catch (e) {}
  });
  document.body.setAttribute("data-math-ready", "true");
</script>
</body>
</html>`

// BuildExamDocument monta o HTML final da prova (seção 28) de UM tipo de
// prova específico: título, o selo "TIPO N" em destaque (requisito de ter
// o tipo bem visível tanto na prova quanto no gabarito) e uma seção por
// questão, na ordem e com as alternativas já na ordem daquele tipo
// (VariantQuestionDetail já resolve isso — ver Repository.LoadVariantQuestions).
func BuildExamDocument(title string, variantNumber int, questions []VariantQuestionDetail) (string, error) {
	var body strings.Builder
	fmt.Fprintf(&body, `<div class="variant-badge">TIPO %d</div>`, variantNumber)
	body.WriteString("<h1>")
	body.WriteString(html.EscapeString(title))
	body.WriteString("</h1>")

	for _, q := range questions {
		statementHTML, err := RenderHTML(q.StatementJSON)
		if err != nil {
			return "", fmt.Errorf("snapshot %d (enunciado): %w", q.SnapshotID, err)
		}
		commandHTML, err := RenderHTML(q.CommandJSON)
		if err != nil {
			return "", fmt.Errorf("snapshot %d (comando): %w", q.SnapshotID, err)
		}

		body.WriteString(`<section class="question">`)
		fmt.Fprintf(&body, `<div class="question-number">Questão %d</div>`, q.PositionInVariant)
		fmt.Fprintf(&body, `<div class="statement">%s</div>`, statementHTML)
		fmt.Fprintf(&body, `<div class="command">%s</div>`, commandHTML)

		body.WriteString(`<ol class="alternatives">`)
		for _, alt := range q.Alternatives {
			altHTML, err := RenderHTML(alt.Content)
			if err != nil {
				return "", fmt.Errorf("snapshot %d alternativa %s: %w", q.SnapshotID, alt.Position, err)
			}
			fmt.Fprintf(&body, `<li>%s</li>`, altHTML)
		}
		body.WriteString(`</ol></section>`)
	}

	return htmlHead + body.String() + htmlTail, nil
}

// BuildAnswerKeyDocument monta o gabarito de UM tipo de prova: o selo "TIPO
// N" e a lista "Q# - Letra" em colunas (column-width no CSS — o próprio
// motor de renderização decide quantas colunas cabem, minimizando o número
// de páginas, idealmente uma só, sem precisar calcular isso aqui).
func BuildAnswerKeyDocument(title string, variantNumber int, questions []VariantQuestionDetail) (string, error) {
	var body strings.Builder
	fmt.Fprintf(&body, `<div class="variant-badge">TIPO %d</div>`, variantNumber)
	body.WriteString("<h1>Gabarito — ")
	body.WriteString(html.EscapeString(title))
	body.WriteString("</h1>")

	body.WriteString(`<div class="answer-key">`)
	for _, q := range questions {
		fmt.Fprintf(&body, `<div class="answer-item">Q%s - %s</div>`, strconv.Itoa(q.PositionInVariant), html.EscapeString(q.CorrectLetter))
	}
	body.WriteString(`</div>`)

	return htmlHead + body.String() + htmlTail, nil
}
