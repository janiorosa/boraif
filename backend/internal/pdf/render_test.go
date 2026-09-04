package pdf

import (
	"strings"
	"testing"
)

// Seção 43: "PDF — criar pelo menos um teste de geração/renderização que
// verifique que um documento válido é produzido." Testar chromedp de
// verdade exigiria um Chromium instalado; o que dá para testar sem isso
// (e o que mais importa ficar correto) é a conversão ProseMirror → HTML.
func TestRenderHTML_EmptyDocument(t *testing.T) {
	out, err := RenderHTML(nil)
	if err != nil {
		t.Fatalf("expected no error for empty document, got %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output for empty document, got %q", out)
	}
}

func TestRenderHTML_ParagraphWithBoldText(t *testing.T) {
	doc := []byte(`{
		"type": "doc",
		"content": [
			{
				"type": "paragraph",
				"content": [
					{"type": "text", "text": "olá "},
					{"type": "text", "text": "mundo", "marks": [{"type": "bold"}]}
				]
			}
		]
	}`)

	out, err := RenderHTML(doc)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}
	if !strings.Contains(out, "olá ") {
		t.Errorf("expected plain text to be present, got %q", out)
	}
	if !strings.Contains(out, "<strong>mundo</strong>") {
		t.Errorf("expected bold text wrapped in <strong>, got %q", out)
	}
}

func TestRenderHTML_EscapesUserText(t *testing.T) {
	doc := []byte(`{
		"type": "doc",
		"content": [
			{"type": "paragraph", "content": [{"type": "text", "text": "<script>alert(1)</script>"}]}
		]
	}`)

	out, err := RenderHTML(doc)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}
	if strings.Contains(out, "<script>") {
		t.Fatalf("expected user text to be HTML-escaped, got %q", out)
	}
}

func TestRenderHTML_BulletList(t *testing.T) {
	doc := []byte(`{
		"type": "doc",
		"content": [
			{"type": "bulletList", "content": [
				{"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "item 1"}]}]}
			]}
		]
	}`)

	out, err := RenderHTML(doc)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}
	if !strings.Contains(out, "<ul>") || !strings.Contains(out, "<li>") || !strings.Contains(out, "item 1") {
		t.Fatalf("expected a rendered bullet list, got %q", out)
	}
}

func TestRenderHTML_InlineMathPlaceholder(t *testing.T) {
	doc := []byte(`{
		"type": "doc",
		"content": [
			{"type": "paragraph", "content": [
				{"type": "inlineMath", "attrs": {"latex": "x^2"}}
			]}
		]
	}`)

	out, err := RenderHTML(doc)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}
	if !strings.Contains(out, `class="math"`) || !strings.Contains(out, `data-latex="x^2"`) {
		t.Fatalf("expected a .math placeholder with the latex source, got %q", out)
	}
}

func TestRenderHTML_UnknownNodeFallsBackToChildren(t *testing.T) {
	doc := []byte(`{
		"type": "doc",
		"content": [
			{"type": "someFutureNode", "content": [
				{"type": "paragraph", "content": [{"type": "text", "text": "não deve sumir"}]}
			]}
		]
	}`)

	out, err := RenderHTML(doc)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}
	if !strings.Contains(out, "não deve sumir") {
		t.Fatalf("expected content of unknown node types to still be rendered, got %q", out)
	}
}

func TestBuildExamDocument_ProducesValidLookingHTML(t *testing.T) {
	questions := []VariantQuestionDetail{
		{
			SnapshotID:        1,
			PositionInVariant: 1,
			StatementJSON:     []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Enunciado"}]}]}`),
			CommandJSON:       []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Comando"}]}]}`),
			Alternatives: []SnapshotAlternative{
				{Position: "A", Content: []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Alt A"}]}]}`), IsCorrect: true},
				{Position: "B", Content: []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Alt B"}]}]}`)},
			},
			CorrectLetter: "A",
		},
	}

	html, err := BuildExamDocument("Prova de Teste", 1, questions)
	if err != nil {
		t.Fatalf("BuildExamDocument failed: %v", err)
	}
	if !strings.HasPrefix(html, "<!doctype html>") {
		t.Error("expected document to start with <!doctype html>")
	}
	if !strings.Contains(html, "Prova de Teste") {
		t.Error("expected the title to appear in the document")
	}
	if !strings.Contains(html, "TIPO 1") {
		t.Error("expected the variant badge to appear in the document")
	}
	if !strings.Contains(html, "Questão 1") {
		t.Error("expected the question number to appear in the document")
	}
	if !strings.Contains(html, "Enunciado") || !strings.Contains(html, "Comando") || !strings.Contains(html, "Alt A") {
		t.Error("expected statement, command and alternatives to appear in the document")
	}
	if !strings.Contains(html, "data-math-ready") {
		t.Error("expected the math-ready marker script used by chromium.go's wait condition")
	}
}

func TestBuildAnswerKeyDocument_ProducesValidLookingHTML(t *testing.T) {
	questions := []VariantQuestionDetail{
		{PositionInVariant: 1, CorrectLetter: "A"},
		{PositionInVariant: 2, CorrectLetter: "C"},
	}

	html, err := BuildAnswerKeyDocument("Prova de Teste", 2, questions)
	if err != nil {
		t.Fatalf("BuildAnswerKeyDocument failed: %v", err)
	}
	if !strings.Contains(html, "TIPO 2") {
		t.Error("expected the variant badge to appear in the answer key")
	}
	if !strings.Contains(html, "Q1 - A") || !strings.Contains(html, "Q2 - C") {
		t.Error("expected each question's answer to appear as 'Q# - Letter'")
	}
}
