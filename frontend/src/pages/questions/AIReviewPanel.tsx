import { useState } from "react";
import { api, ApiError } from "../../api/client";
import type { AlternativePosition } from "./types";

interface AlternativeInput {
  position: AlternativePosition;
  text: string;
  isCorrect: boolean;
}

interface ReviewResult {
  summary: string;
  issues: string[];
  suggestions: string[];
}

type Target = "statement" | "command" | "alternatives" | "full";

interface Props {
  questionId: number;
  gradeYearName: string;
  difficultyName: string;
  getStatementText: () => string;
  getCommandText: () => string;
  getAlternatives: () => AlternativeInput[];
}

const TARGET_LABELS: Record<Target, string> = {
  statement: "Revisar enunciado",
  command: "Revisar comando",
  alternatives: "Revisar alternativas",
  full: "Revisar questão inteira",
};

// Assistente de revisão por IA (seção 16): nunca gera nem substitui
// conteúdo — só devolve críticas e sugestões; o professor decide o que
// aceitar. Usa a própria API Key do professor (seção 17, configurada em
// "Minha conta"); se não houver chave cadastrada, o backend explica isso
// na mensagem de erro.
export function AIReviewPanel({
  questionId,
  gradeYearName,
  difficultyName,
  getStatementText,
  getCommandText,
  getAlternatives,
}: Props) {
  const [loadingTarget, setLoadingTarget] = useState<Target | null>(null);
  const [result, setResult] = useState<ReviewResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleReview(target: Target) {
    setLoadingTarget(target);
    setError(null);
    setResult(null);
    try {
      const response = await api.post<ReviewResult>(`/api/questions/${questionId}/ai/review`, {
        target,
        gradeYear: gradeYearName,
        difficulty: difficultyName,
        statement: getStatementText(),
        command: getCommandText(),
        alternatives: getAlternatives(),
      });
      setResult(response);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível obter a análise da IA.");
    } finally {
      setLoadingTarget(null);
    }
  }

  return (
    <div style={{ marginTop: 24, border: "1px solid #ccc", borderRadius: 4, padding: 12, maxWidth: 760 }}>
      <h2 style={{ marginTop: 0 }}>Assistente de revisão por IA</h2>
      <p style={{ color: "#666", fontSize: 14 }}>
        A IA só sugere e critica — ela nunca altera sua questão. Você decide o que aceitar.
      </p>
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
        {(Object.keys(TARGET_LABELS) as Target[]).map((target) => (
          <button key={target} type="button" onClick={() => handleReview(target)} disabled={loadingTarget !== null}>
            {loadingTarget === target ? "Analisando..." : TARGET_LABELS[target]}
          </button>
        ))}
      </div>

      {error && <p style={{ color: "crimson" }}>{error}</p>}

      {result && (
        <div style={{ marginTop: 12 }}>
          <p>{result.summary}</p>
          {result.issues.length > 0 && (
            <>
              <strong>Problemas apontados:</strong>
              <ul>
                {result.issues.map((issue, i) => (
                  <li key={i}>{issue}</li>
                ))}
              </ul>
            </>
          )}
          {result.suggestions.length > 0 && (
            <>
              <strong>Sugestões:</strong>
              <ul>
                {result.suggestions.map((s, i) => (
                  <li key={i}>{s}</li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}
    </div>
  );
}
