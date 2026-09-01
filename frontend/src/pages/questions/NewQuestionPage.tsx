import { useEffect, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { AppLayout } from "../../components/AppLayout";
import { api, ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import type { Discipline } from "../../types";
import type { Subject } from "../subjects/types";
import type { Difficulty, GradeYear, QuestionDetail } from "./types";
import { emptyDoc } from "./prosemirror";
import { ALL_POSITIONS } from "./constants";

// Fluxo da seção 37: escolher ano, assunto e dificuldade, e já começar a
// trabalhar na questão — a disciplina já vem determinada pelo usuário
// (a própria, para ELABORADOR) e o status inicial é sempre RASCUNHO.
// O conteúdo (enunciado/comando/alternativas) começa vazio e é preenchido
// na tela de edição; o corpo da questão em si (o editor rico) chega na Fase 5.
export function NewQuestionPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === "ADMIN";
  const navigate = useNavigate();

  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [gradeYears, setGradeYears] = useState<GradeYear[]>([]);
  const [difficulties, setDifficulties] = useState<Difficulty[]>([]);

  const [disciplineId, setDisciplineId] = useState<number | "">(isAdmin ? "" : user?.disciplineId ?? "");
  const [subjectId, setSubjectId] = useState<number | "">("");
  const [gradeYearId, setGradeYearId] = useState<number | "">("");
  const [difficultyId, setDifficultyId] = useState<number | "">("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.get<Discipline[]>("/api/disciplines"),
      api.get<GradeYear[]>("/api/grade-years"),
      api.get<Difficulty[]>("/api/difficulties"),
    ]).then(([d, gy, df]) => {
      if (cancelled) return;
      setDisciplines(d);
      setGradeYears(gy);
      setDifficulties(df);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (disciplineId === "") {
      setSubjects([]);
      return;
    }
    let cancelled = false;
    api.get<Subject[]>(`/api/subjects?disciplineId=${disciplineId}`).then((list) => {
      if (!cancelled) setSubjects(list);
    });
    return () => {
      cancelled = true;
    };
  }, [disciplineId]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (subjectId === "" || gradeYearId === "" || difficultyId === "" || (isAdmin && disciplineId === "")) return;

    setSubmitting(true);
    try {
      const created = await api.post<QuestionDetail>("/api/questions", {
        disciplineId: isAdmin ? disciplineId : undefined,
        subjectId,
        gradeYearId,
        difficultyId,
        statement: emptyDoc(),
        command: emptyDoc(),
        alternatives: ALL_POSITIONS.map((position) => ({
          position,
          content: emptyDoc(),
          isCorrect: position === "A",
        })),
      });
      navigate(`/questoes/${created.id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível criar a questão.");
      setSubmitting(false);
    }
  }

  return (
    <AppLayout>
      <h1>Nova questão</h1>
      <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 10, maxWidth: 360 }}>
        {isAdmin && (
          <label>
            Disciplina
            <select
              value={disciplineId}
              onChange={(e) => {
                setDisciplineId(e.target.value === "" ? "" : Number(e.target.value));
                setSubjectId("");
              }}
              required
              style={{ width: "100%" }}
            >
              <option value="" disabled>
                Selecione...
              </option>
              {disciplines.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
        )}

        <label>
          Assunto
          <select
            value={subjectId}
            onChange={(e) => setSubjectId(e.target.value === "" ? "" : Number(e.target.value))}
            required
            disabled={disciplineId === ""}
            style={{ width: "100%" }}
          >
            <option value="" disabled>
              Selecione...
            </option>
            {subjects.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </label>

        <label>
          Ano
          <select
            value={gradeYearId}
            onChange={(e) => setGradeYearId(e.target.value === "" ? "" : Number(e.target.value))}
            required
            style={{ width: "100%" }}
          >
            <option value="" disabled>
              Selecione...
            </option>
            {gradeYears.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </select>
        </label>

        <label>
          Dificuldade
          <select
            value={difficultyId}
            onChange={(e) => setDifficultyId(e.target.value === "" ? "" : Number(e.target.value))}
            required
            style={{ width: "100%" }}
          >
            <option value="" disabled>
              Selecione...
            </option>
            {difficulties.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </label>

        {error && <p style={{ color: "crimson" }}>{error}</p>}

        <button type="submit" disabled={submitting}>
          {submitting ? "Criando..." : "Criar e começar a escrever"}
        </button>
      </form>
    </AppLayout>
  );
}
