import { useState, type FormEvent } from "react";
import type { Discipline } from "../../types";
import type { Difficulty, GradeYear } from "../questions/types";
import type { Subject } from "../subjects/types";
import type { Configuration, QuotaRule } from "./types";

interface Props {
  configuration: Configuration;
  disciplines: Discipline[];
  subjects: Subject[];
  gradeYears: GradeYear[];
  difficulties: Difficulty[];
  onSubmit: (updated: { totalQuestions: number; gradeYearIds: number[]; quotaRules: QuotaRule[] }) => void;
  submitting: boolean;
  error: string | null;
}

// Editor de configuração compartilhado pelo caderno (seção 22/23) e pela
// configuração padrão (seção 22) — mesmo formato exato dos dois lados da
// API, então um único componente evita duplicar a tela.
export function ConfigurationEditor({
  configuration,
  disciplines,
  subjects,
  gradeYears,
  difficulties,
  onSubmit,
  submitting,
  error,
}: Props) {
  const [totalQuestions, setTotalQuestions] = useState(configuration.totalQuestions);
  const [gradeYearIds, setGradeYearIds] = useState<number[]>(configuration.gradeYearIds);
  const [quotaRules, setQuotaRules] = useState<QuotaRule[]>(configuration.quotaRules);

  const readOnly = configuration.isFrozen;
  const sum = quotaRules.reduce((acc, q) => acc + (q.quantity || 0), 0);

  function toggleGradeYear(id: number) {
    setGradeYearIds((prev) => (prev.includes(id) ? prev.filter((y) => y !== id) : [...prev, id]));
  }

  function addQuotaRule() {
    setQuotaRules((prev) => [...prev, { disciplineId: disciplines[0]?.id ?? 0, quantity: 1 }]);
  }

  function updateQuotaRule(index: number, patch: Partial<QuotaRule>) {
    setQuotaRules((prev) => prev.map((q, i) => (i === index ? { ...q, ...patch } : q)));
  }

  function removeQuotaRule(index: number) {
    setQuotaRules((prev) => prev.filter((_, i) => i !== index));
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit({ totalQuestions, gradeYearIds, quotaRules });
  }

  return (
    <form onSubmit={handleSubmit}>
      {readOnly && (
        <p style={{ color: "#a60", fontWeight: "bold" }}>
          Esta configuração já foi congelada (prova gerada) e não pode mais ser alterada.
        </p>
      )}
      <fieldset disabled={readOnly} style={{ border: "none", padding: 0, margin: 0 }}>
        <label>
          Total de questões
          <input
            type="number"
            min={1}
            value={totalQuestions}
            onChange={(e) => setTotalQuestions(Number(e.target.value))}
            style={{ display: "block", width: 120 }}
          />
        </label>

        <fieldset style={{ marginTop: 16 }}>
          <legend>Anos</legend>
          {gradeYears.map((g) => (
            <label key={g.id} style={{ marginRight: 16 }}>
              <input type="checkbox" checked={gradeYearIds.includes(g.id)} onChange={() => toggleGradeYear(g.id)} /> {g.name}
            </label>
          ))}
        </fieldset>

        <fieldset style={{ marginTop: 16 }}>
          <legend>Cotas por disciplina (opcionalmente por assunto e/ou dificuldade)</legend>
          <table style={{ borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ textAlign: "left" }}>
                <th>Disciplina</th>
                <th>Assunto</th>
                <th>Dificuldade</th>
                <th>Quantidade</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {quotaRules.map((q, i) => (
                <tr key={i}>
                  <td>
                    <select
                      value={q.disciplineId}
                      onChange={(e) => updateQuotaRule(i, { disciplineId: Number(e.target.value), subjectId: null })}
                    >
                      {disciplines.map((d) => (
                        <option key={d.id} value={d.id}>
                          {d.name}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td>
                    <select
                      value={q.subjectId ?? ""}
                      onChange={(e) =>
                        updateQuotaRule(i, { subjectId: e.target.value === "" ? null : Number(e.target.value) })
                      }
                    >
                      <option value="">(qualquer)</option>
                      {subjects
                        .filter((s) => s.disciplineId === q.disciplineId)
                        .map((s) => (
                          <option key={s.id} value={s.id}>
                            {s.name}
                          </option>
                        ))}
                    </select>
                  </td>
                  <td>
                    <select
                      value={q.difficultyId ?? ""}
                      onChange={(e) =>
                        updateQuotaRule(i, { difficultyId: e.target.value === "" ? null : Number(e.target.value) })
                      }
                    >
                      <option value="">(qualquer)</option>
                      {difficulties.map((d) => (
                        <option key={d.id} value={d.id}>
                          {d.name}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td>
                    <input
                      type="number"
                      min={1}
                      value={q.quantity}
                      onChange={(e) => updateQuotaRule(i, { quantity: Number(e.target.value) })}
                      style={{ width: 60 }}
                    />
                  </td>
                  <td>
                    <button type="button" onClick={() => removeQuotaRule(i)}>
                      Remover
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <button type="button" onClick={addQuotaRule} style={{ marginTop: 8 }}>
            + Adicionar cota
          </button>
          <p style={{ color: sum === totalQuestions ? "green" : "crimson" }}>
            Soma das cotas: {sum} — Total de questões: {totalQuestions}
            {sum !== totalQuestions && " (precisam ser iguais para salvar)"}
          </p>
        </fieldset>

        {error && <p style={{ color: "crimson" }}>{error}</p>}
        <button type="submit" disabled={submitting || sum !== totalQuestions} style={{ marginTop: 8 }}>
          {submitting ? "Salvando..." : "Salvar configuração"}
        </button>
      </fieldset>
    </form>
  );
}
