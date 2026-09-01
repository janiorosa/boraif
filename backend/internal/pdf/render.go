package pdf

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// proseMirrorNode é um nó genérico do documento ProseMirror/TipTap (seção 8).
// O backend continua sem conhecer a estrutura completa do editor para fins
// de CRUD (seções 4/5 — jsonb opaco); este arquivo é a única exceção
// deliberada, porque gerar o PDF exige produzir HTML a partir do JSON em
// algum lugar (seção 28: "Questões estruturadas → HTML → ... → PDF"), e o
// subconjunto de nós/marcas é o mesmo, fixo, conjunto de extensões que o
// BoraIF habilita em frontend/src/components/editor/RichTextEditor.tsx.
type proseMirrorNode struct {
	Type    string            `json:"type"`
	Text    string            `json:"text,omitempty"`
	Attrs   map[string]any    `json:"attrs,omitempty"`
	Content []proseMirrorNode `json:"content,omitempty"`
	Marks   []proseMirrorMark `json:"marks,omitempty"`
}

type proseMirrorMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// RenderHTML converte um documento ProseMirror/TipTap em HTML para a prova
// impressa. Nós desconhecidos têm o conteúdo interno renderizado do mesmo
// jeito (em vez de descartados), para nunca perder texto do professor
// silenciosamente caso o editor ganhe uma extensão nova no futuro.
func RenderHTML(doc json.RawMessage) (string, error) {
	if len(doc) == 0 {
		return "", nil
	}
	var root proseMirrorNode
	if err := json.Unmarshal(doc, &root); err != nil {
		return "", fmt.Errorf("decoding ProseMirror document: %w", err)
	}
	var b strings.Builder
	renderNodes(&b, root.Content)
	return b.String(), nil
}

func renderNodes(b *strings.Builder, nodes []proseMirrorNode) {
	for _, n := range nodes {
		renderNode(b, n)
	}
}

func renderNode(b *strings.Builder, n proseMirrorNode) {
	switch n.Type {
	case "text":
		b.WriteString(applyMarks(html.EscapeString(n.Text), n.Marks))
	case "paragraph":
		b.WriteString(`<p style="text-align:` + textAlign(n.Attrs) + `">`)
		renderNodes(b, n.Content)
		b.WriteString(`</p>`)
	case "hardBreak":
		b.WriteString(`<br/>`)
	case "bulletList":
		b.WriteString(`<ul>`)
		renderNodes(b, n.Content)
		b.WriteString(`</ul>`)
	case "orderedList":
		b.WriteString(`<ol>`)
		renderNodes(b, n.Content)
		b.WriteString(`</ol>`)
	case "listItem":
		b.WriteString(`<li>`)
		renderNodes(b, n.Content)
		b.WriteString(`</li>`)
	case "blockquote":
		b.WriteString(`<blockquote>`)
		renderNodes(b, n.Content)
		b.WriteString(`</blockquote>`)
	case "codeBlock":
		b.WriteString(`<pre><code>`)
		renderNodes(b, n.Content)
		b.WriteString(`</code></pre>`)
	case "horizontalRule":
		b.WriteString(`<hr/>`)
	case "image":
		src, _ := n.Attrs["src"].(string)
		alt, _ := n.Attrs["alt"].(string)
		b.WriteString(`<img src="` + html.EscapeString(src) + `" alt="` + html.EscapeString(alt) + `" />`)
	case "table":
		b.WriteString(`<table>`)
		renderNodes(b, n.Content)
		b.WriteString(`</table>`)
	case "tableRow":
		b.WriteString(`<tr>`)
		renderNodes(b, n.Content)
		b.WriteString(`</tr>`)
	case "tableCell":
		b.WriteString(`<td>`)
		renderNodes(b, n.Content)
		b.WriteString(`</td>`)
	case "tableHeader":
		b.WriteString(`<th>`)
		renderNodes(b, n.Content)
		b.WriteString(`</th>`)
	case "inlineMath":
		writeMath(b, n.Attrs, false)
	case "blockMath":
		writeMath(b, n.Attrs, true)
	default:
		renderNodes(b, n.Content)
	}
}

// writeMath deixa um placeholder que o próprio KaTeX, carregado na página
// que o Chromium abre para imprimir, substitui pela fórmula renderizada
// (seção 12) — nenhuma fórmula é gerada como PDF individualmente (seção 11).
func writeMath(b *strings.Builder, attrs map[string]any, block bool) {
	latex, _ := attrs["latex"].(string)
	tag := "span"
	if block {
		tag = "div"
	}
	fmt.Fprintf(b, `<%s class="math" data-latex="%s"></%s>`, tag, html.EscapeString(latex), tag)
}

func textAlign(attrs map[string]any) string {
	align, _ := attrs["textAlign"].(string)
	if align == "" {
		return "left"
	}
	return align
}

func applyMarks(text string, marks []proseMirrorMark) string {
	for _, m := range marks {
		switch m.Type {
		case "bold":
			text = "<strong>" + text + "</strong>"
		case "italic":
			text = "<em>" + text + "</em>"
		case "underline":
			text = "<u>" + text + "</u>"
		case "strike":
			text = "<s>" + text + "</s>"
		case "subscript":
			text = "<sub>" + text + "</sub>"
		case "superscript":
			text = "<sup>" + text + "</sup>"
		case "link":
			href, _ := m.Attrs["href"].(string)
			text = `<a href="` + html.EscapeString(href) + `">` + text + `</a>`
		}
	}
	return text
}
