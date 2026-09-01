import { useEffect, useState, type FormEvent } from "react";
import { AppLayout } from "../../components/AppLayout";
import { api, ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import type { Discipline } from "../../types";
import type { SimilarSubject, Subject } from "./types";

// undefined = sem filtro (todas as disciplinas) — só usado pelo ADMIN;
// o ELABORADOR sempre trabalha dentro da própria disciplina (seção 15).
export function SubjectsPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === "ADMIN";
  const ownDisciplineId = user?.disciplineId ?? undefined;

  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [loading, setLoading] = useState(true);
  const [filterDisciplineId, setFilterDisciplineId] = useState<number | "">("");

  const [newName, setNewName] = useState("");
  const [newDisciplineId, setNewDisciplineId] = useState<number | "">("");
  const [pendingConfirmation, setPendingConfirmation] = useState<{
    disciplineId: number;
    name: string;
    similar: SimilarSubject[];
  } | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [editingId, setEditingId] = useState<number | null>(null);
  const [editingName, setEditingName] = useState("");
  const [rowError, setRowError] = useState<string | null>(null);

  const effectiveDisciplineId = isAdmin ? (filterDisciplineId === "" ? undefined : filterDisciplineId) : ownDisciplineId;

  useEffect(() => {
    let cancelled = false;
    api
      .get<Discipline[]>("/api/disciplines")
      .then((list) => {
        if (!cancelled) setDisciplines(list);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const query = effectiveDisciplineId !== undefined ? `?disciplineId=${effectiveDisciplineId}` : "";
    api
      .get<Subject[]>(`/api/subjects${query}`)
      .then((list) => {
        if (!cancelled) setSubjects(list);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [effectiveDisciplineId]);

  function disciplineName(id: number): string {
    return disciplines.find((d) => d.id === id)?.name ?? "—";
  }

  async function submitCreate(disciplineId: number, name: string, confirmDuplicate: boolean) {
    const created = await api.post<Subject>("/api/subjects", { disciplineId, name, confirmDuplicate });
    setSubjects((prev) => [...prev, created].sort((a, b) => a.name.localeCompare(b.name)));
    setNewName("");
    setPendingConfirmation(null);
  }

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setFormError(null);

    const disciplineId = isAdmin ? Number(newDisciplineId) : ownDisciplineId;
    const name = newName.trim();
    if (!disciplineId || !name) return;

    setSubmitting(true);
    try {
      await submitCreate(disciplineId, name, false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409 && isSimilarConflict(err.body)) {
        setPendingConfirmation({ disciplineId, name, similar: err.body.similar });
      } else {
        setFormError(err instanceof ApiError ? err.message : "Não foi possível criar o assunto.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  async function handleConfirmDuplicate() {
    if (!pendingConfirmation) return;
    setSubmitting(true);
    setFormError(null);
    try {
      await submitCreate(pendingConfirmation.disciplineId, pendingConfirmation.name, true);
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Não foi possível criar o assunto.");
    } finally {
      setSubmitting(false);
    }
  }

  function startEdit(subject: Subject) {
    setRowError(null);
    setEditingId(subject.id);
    setEditingName(subject.name);
  }

  async function saveEdit(id: number) {
    setRowError(null);
    const name = editingName.trim();
    if (!name) return;
    try {
      await api.put(`/api/subjects/${id}`, { name });
      setSubjects((prev) => prev.map((s) => (s.id === id ? { ...s, name } : s)));
      setEditingId(null);
    } catch (err) {
      setRowError(err instanceof ApiError ? err.message : "Não foi possível salvar.");
    }
  }

  async function handleDelete(subject: Subject) {
    if (!confirm(`Excluir o assunto "${subject.name}"?`)) return;
    setRowError(null);
    try {
      await api.delete(`/api/subjects/${subject.id}`);
      setSubjects((prev) => prev.filter((s) => s.id !== subject.id));
    } catch (err) {
      setRowError(err instanceof ApiError ? err.message : "Não foi possível excluir.");
    }
  }

  return (
    <AppLayout>
      <h1>Assuntos</h1>

      {isAdmin && (
        <label>
          Filtrar por disciplina:{" "}
          <select
            value={filterDisciplineId}
            onChange={(e) => setFilterDisciplineId(e.target.value === "" ? "" : Number(e.target.value))}
          >
            <option value="">Todas</option>
            {disciplines.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </label>
      )}

      {rowError && <p style={{ color: "crimson" }}>{rowError}</p>}

      {loading ? (
        <p>Carregando...</p>
      ) : (
        <table style={{ width: "100%", borderCollapse: "collapse", marginTop: 12 }}>
          <thead>
            <tr style={{ textAlign: "left", borderBottom: "1px solid #ccc" }}>
              <th>Nome</th>
              {isAdmin && <th>Disciplina</th>}
              {isAdmin && <th />}
            </tr>
          </thead>
          <tbody>
            {subjects.map((s) => (
              <tr key={s.id} style={{ borderBottom: "1px solid #eee" }}>
                <td>
                  {editingId === s.id ? (
                    <input value={editingName} onChange={(e) => setEditingName(e.target.value)} />
                  ) : (
                    s.name
                  )}
                </td>
                {isAdmin && <td>{disciplineName(s.disciplineId)}</td>}
                {isAdmin && (
                  <td>
                    {editingId === s.id ? (
                      <>
                        <button onClick={() => saveEdit(s.id)}>Salvar</button>
                        <button onClick={() => setEditingId(null)}>Cancelar</button>
                      </>
                    ) : (
                      <>
                        <button onClick={() => startEdit(s)}>Editar</button>
                        <button onClick={() => handleDelete(s)}>Excluir</button>
                      </>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h2 style={{ marginTop: 32 }}>Novo assunto</h2>
      <form onSubmit={handleCreate} style={{ display: "flex", gap: 8, alignItems: "flex-end", maxWidth: 480 }}>
        {isAdmin && (
          <label>
            Disciplina
            <select
              value={newDisciplineId}
              onChange={(e) => setNewDisciplineId(e.target.value === "" ? "" : Number(e.target.value))}
              required
              style={{ display: "block" }}
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
        <label style={{ flex: 1 }}>
          Nome
          <input value={newName} onChange={(e) => setNewName(e.target.value)} required style={{ width: "100%" }} />
        </label>
        <button type="submit" disabled={submitting}>
          Criar
        </button>
      </form>

      {formError && <p style={{ color: "crimson" }}>{formError}</p>}

      {pendingConfirmation && (
        <div style={{ border: "1px solid #e0a000", padding: 12, marginTop: 8, maxWidth: 480 }}>
          <p>
            Já existe(m) assunto(s) parecido(s) com "<strong>{pendingConfirmation.name}</strong>" nesta disciplina:
          </p>
          <ul>
            {pendingConfirmation.similar.map((s) => (
              <li key={s.id}>{s.name}</li>
            ))}
          </ul>
          <button onClick={handleConfirmDuplicate} disabled={submitting}>
            Criar mesmo assim
          </button>
          <button onClick={() => setPendingConfirmation(null)} disabled={submitting}>
            Cancelar
          </button>
        </div>
      )}
    </AppLayout>
  );
}

function isSimilarConflict(body: unknown): body is { similar: SimilarSubject[] } {
  return typeof body === "object" && body !== null && Array.isArray((body as { similar?: unknown }).similar);
}
