import { useState } from "react";
import katex from "katex";

interface Props {
  onInsert: (latex: string) => void;
  onClose: () => void;
}

// Snippets comuns para quem não conhece LaTeX (seção 11.1); quem conhece
// LaTeX digita direto na caixa de texto (seção 11.2). O preview usa o mesmo
// KaTeX que vai renderizar a fórmula na questão e, mais tarde, no PDF.
const SNIPPETS: { label: string; insert: string }[] = [
  { label: "a⁄b", insert: "\\frac{a}{b}" },
  { label: "√x", insert: "\\sqrt{x}" },
  { label: "xⁿ", insert: "x^{n}" },
  { label: "xₙ", insert: "x_{n}" },
  { label: "±", insert: "\\pm" },
  { label: "Σ", insert: "\\sum_{i=1}^{n}" },
  { label: "∫", insert: "\\int" },
  { label: "∞", insert: "\\infty" },
  { label: "π", insert: "\\pi" },
  { label: "α", insert: "\\alpha" },
  { label: "β", insert: "\\beta" },
  { label: "Δ", insert: "\\Delta" },
  { label: "θ", insert: "\\theta" },
  { label: "≤", insert: "\\leq" },
  { label: "≥", insert: "\\geq" },
  { label: "≠", insert: "\\neq" },
];

const TEXTAREA_ID = "formula-latex-input";

export function FormulaDialog({ onInsert, onClose }: Props) {
  const [latex, setLatex] = useState("");

  let previewHtml = "";
  let previewError: string | null = null;
  if (latex.trim()) {
    try {
      previewHtml = katex.renderToString(latex, { throwOnError: true });
    } catch {
      previewError = "LaTeX inválido";
    }
  }

  function insertAtCursor(snippet: string) {
    const textarea = document.getElementById(TEXTAREA_ID) as HTMLTextAreaElement | null;
    if (!textarea) {
      setLatex((prev) => prev + snippet);
      return;
    }
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const next = latex.slice(0, start) + snippet + latex.slice(end);
    setLatex(next);
    requestAnimationFrame(() => {
      textarea.focus();
      textarea.selectionStart = textarea.selectionEnd = start + snippet.length;
    });
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.3)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
      }}
      onClick={onClose}
    >
      <div style={{ background: "white", padding: 16, borderRadius: 4, width: 420 }} onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>Inserir fórmula</h3>

        <div style={{ display: "flex", flexWrap: "wrap", gap: 4, marginBottom: 8 }}>
          {SNIPPETS.map((s) => (
            <button key={s.label} type="button" onClick={() => insertAtCursor(s.insert)}>
              {s.label}
            </button>
          ))}
        </div>

        <textarea
          id={TEXTAREA_ID}
          value={latex}
          onChange={(e) => setLatex(e.target.value)}
          rows={3}
          placeholder="Digite LaTeX, ex.: \frac{-b \pm \sqrt{b^2-4ac}}{2a}"
          style={{ width: "100%" }}
        />

        <div style={{ margin: "8px 0", minHeight: 32, padding: 4, border: "1px solid #eee" }}>
          {previewError ? (
            <span style={{ color: "crimson" }}>{previewError}</span>
          ) : (
            <span dangerouslySetInnerHTML={{ __html: previewHtml }} />
          )}
        </div>

        <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
          <button type="button" onClick={onClose}>
            Cancelar
          </button>
          <button type="button" disabled={!latex.trim() || !!previewError} onClick={() => onInsert(latex)}>
            Inserir
          </button>
        </div>
      </div>
    </div>
  );
}
