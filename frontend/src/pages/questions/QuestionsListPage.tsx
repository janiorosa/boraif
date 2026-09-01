import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { AppLayout } from "../../components/AppLayout";
import { api } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import type { Discipline } from "../../types";
import type { Difficulty, GradeYear, QuestionStatus, QuestionSummary } from "./types";
import type { Subject } from "../subjects/types";

const PAGE_SIZE = 20;

export function QuestionsListPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === "ADMIN";

  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [gradeYears, setGradeYears] = useState<GradeYear[]>([]);
  const [difficulties, setDifficulties] = useState<Difficulty[]>([]);
  const [statuses, setStatuses] = useState<QuestionStatus[]>([]);

  const [items, setItems] = useState<QuestionSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [disciplineId, setDisciplineId] = useState<number | "">("");
  const [subjectId, setSubjectId] = useState<number | "">("");
  const [gradeYearId, setGradeYearId] = useState<number | "">("");
  const [difficultyId, setDifficultyId] = useState<number | "">("");
  const [statusId, setStatusId] = useState<number | "">("");
  const [page, setPage] = useState(1);

  // debounce da busca por texto — evita uma requisição por tecla (seção 19).
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 400);
    return () => clearTimeout(timer);
  }, [search]);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.get<Discipline[]>("/api/disciplines"),
      api.get<GradeYear[]>("/api/grade-years"),
      api.get<Difficulty[]>("/api/difficulties"),
      api.get<QuestionStatus[]>("/api/question-statuses"),
    ]).then(([d, gy, df, st]) => {
      if (cancelled) return;
      setDisciplines(d);
      setGradeYears(gy);
      setDifficulties(df);
      setStatuses(st);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // Lista de assuntos depende da disciplina em foco (a própria, para
  // ELABORADOR; a selecionada no filtro, para ADMIN).
  const effectiveDisciplineId = isAdmin ? disciplineId : (user?.disciplineId ?? "");
  useEffect(() => {
    if (effectiveDisciplineId === "") {
      setSubjects([]);
      return;
    }
    let cancelled = false;
    api.get<Subject[]>(`/api/subjects?disciplineId=${effectiveDisciplineId}`).then((list) => {
      if (!cancelled) setSubjects(list);
    });
    return () => {
      cancelled = true;
    };
  }, [effectiveDisciplineId]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const params = new URLSearchParams();
    if (debouncedSearch) params.set("search", debouncedSearch);
    if (isAdmin && disciplineId !== "") params.set("disciplineId", String(disciplineId));
    if (subjectId !== "") params.set("subjectId", String(subjectId));
    if (gradeYearId !== "") params.set("gradeYearId", String(gradeYearId));
    if (difficultyId !== "") params.set("difficultyId", String(difficultyId));
    if (statusId !== "") params.set("statusId", String(statusId));
    params.set("page", String(page));
    params.set("pageSize", String(PAGE_SIZE));

    api
      .get<{ items: QuestionSummary[]; total: number }>(`/api/questions?${params.toString()}`)
      .then((result) => {
        if (cancelled) return;
        setItems(result.items);
        setTotal(result.total);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [debouncedSearch, isAdmin, disciplineId, subjectId, gradeYearId, difficultyId, statusId, page]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <AppLayout>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>Questões</h1>
        <Link to="/questoes/nova">
          <button>Nova questão</button>
        </Link>
      </div>

      <div style={{ display: "flex", gap: 8, flexWrap: "wrap", margin: "12px 0" }}>
        <input
          placeholder="Buscar..."
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
        />
        {isAdmin && (
          <select
            value={disciplineId}
            onChange={(e) => {
              setDisciplineId(e.target.value === "" ? "" : Number(e.target.value));
              setSubjectId("");
              setPage(1);
            }}
          >
            <option value="">Todas as disciplinas</option>
            {disciplines.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        )}
        <select
          value={subjectId}
          onChange={(e) => {
            setSubjectId(e.target.value === "" ? "" : Number(e.target.value));
            setPage(1);
          }}
        >
          <option value="">Todos os assuntos</option>
          {subjects.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
        <select
          value={gradeYearId}
          onChange={(e) => {
            setGradeYearId(e.target.value === "" ? "" : Number(e.target.value));
            setPage(1);
          }}
        >
          <option value="">Todos os anos</option>
          {gradeYears.map((g) => (
            <option key={g.id} value={g.id}>
              {g.name}
            </option>
          ))}
        </select>
        <select
          value={difficultyId}
          onChange={(e) => {
            setDifficultyId(e.target.value === "" ? "" : Number(e.target.value));
            setPage(1);
          }}
        >
          <option value="">Todas as dificuldades</option>
          {difficulties.map((d) => (
            <option key={d.id} value={d.id}>
              {d.name}
            </option>
          ))}
        </select>
        <select
          value={statusId}
          onChange={(e) => {
            setStatusId(e.target.value === "" ? "" : Number(e.target.value));
            setPage(1);
          }}
        >
          <option value="">Todos os status</option>
          {statuses.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
      </div>

      {loading ? (
        <p>Carregando...</p>
      ) : (
        <>
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ textAlign: "left", borderBottom: "1px solid #ccc" }}>
                <th>Disciplina</th>
                <th>Assunto</th>
                <th>Ano</th>
                <th>Dificuldade</th>
                <th>Status</th>
                <th>Autor</th>
                <th>Atualizada em</th>
                <th>Rev.</th>
              </tr>
            </thead>
            <tbody>
              {items.map((q) => (
                <tr key={q.id} style={{ borderBottom: "1px solid #eee" }}>
                  <td>{q.disciplineName}</td>
                  <td>{q.subjectName}</td>
                  <td>{q.gradeYearName}</td>
                  <td>{q.difficultyName}</td>
                  <td>{q.statusName}</td>
                  <td>{q.authorName}</td>
                  <td>{new Date(q.updatedAt).toLocaleString("pt-BR")}</td>
                  <td>{q.revisionNumber}</td>
                  <td>
                    <Link to={`/questoes/${q.id}`}>Abrir</Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div style={{ display: "flex", gap: 8, alignItems: "center", marginTop: 12 }}>
            <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
              Anterior
            </button>
            <span>
              Página {page} de {totalPages} ({total} questões)
            </span>
            <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
              Próxima
            </button>
          </div>
        </>
      )}
    </AppLayout>
  );
}
