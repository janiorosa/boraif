package pdf

import (
	"fmt"
	"html"
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

// BuildDocument monta o HTML final da prova (seção 28): título, e uma seção
// por questão do snapshot, numerada sequencialmente (seção 26) — a mesma
// numeração 1..N já gravada em position_in_booklet.
func BuildDocument(title string, snapshots []Snapshot) (string, error) {
	var body strings.Builder
	body.WriteString("<h1>")
	body.WriteString(html.EscapeString(title))
	body.WriteString("</h1>")

	for _, s := range snapshots {
		statementHTML, err := RenderHTML(s.StatementJSON)
		if err != nil {
			return "", fmt.Errorf("snapshot %d (enunciado): %w", s.ID, err)
		}
		commandHTML, err := RenderHTML(s.CommandJSON)
		if err != nil {
			return "", fmt.Errorf("snapshot %d (comando): %w", s.ID, err)
		}

		body.WriteString(`<section class="question">`)
		fmt.Fprintf(&body, `<div class="question-number">Questão %d</div>`, s.PositionInBooklet)
		fmt.Fprintf(&body, `<div class="statement">%s</div>`, statementHTML)
		fmt.Fprintf(&body, `<div class="command">%s</div>`, commandHTML)

		body.WriteString(`<ol class="alternatives">`)
		for _, alt := range s.Alternatives {
			altHTML, err := RenderHTML(alt.Content)
			if err != nil {
				return "", fmt.Errorf("snapshot %d alternativa %s: %w", s.ID, alt.Position, err)
			}
			fmt.Fprintf(&body, `<li>%s</li>`, altHTML)
		}
		body.WriteString(`</ol></section>`)
	}

	return htmlHead + body.String() + htmlTail, nil
}
