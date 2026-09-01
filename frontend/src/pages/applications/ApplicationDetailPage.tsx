import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { AppLayout } from "../../components/AppLayout";
import { api, ApiError } from "../../api/client";
import type { Application, ApplicationStatus, Booklet } from "./types";

export function ApplicationDetailPage() {
  const { id } = useParams<{ id: string }>();
  const applicationId = Number(id);

  const [application, setApplication] = useState<Application | null>(null);
  const [booklets, setBooklets] = useState<Booklet[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [newBookletName, setNewBookletName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const [savingStatus, setSavingStatus] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.get<Application>(`/api/applications/${applicationId}`),
      api.get<Booklet[]>(`/api/applications/${applicationId}/booklets`),
    ])
      .then(([app, list]) => {
        if (cancelled) return;
        setApplication(app);
        setBooklets(list);
      })
      .catch((err) => {
        if (!cancelled) setLoadError(err instanceof ApiError ? err.message : "Não foi possível carregar a aplicação.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [applicationId]);

  async function handleCreateBooklet(e: FormEvent) {
    e.preventDefault();
    setCreating(true);
    setCreateError(null);
    try {
      const created = await api.post<Booklet>(`/api/applications/${applicationId}/booklets`, { name: newBookletName });
      setBooklets((prev) => [...prev, created]);
      setNewBookletName("");
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : "Não foi possível criar o caderno.");
    } finally {
      setCreating(false);
    }
  }

  async function handleStatusChange(newStatus: ApplicationStatus) {
    if (!application) return;
    setSavingStatus(true);
    try {
      const updated = await api.put<Application>(`/api/applications/${applicationId}`, {
        name: application.name,
        description: application.description,
        status: newStatus,
      });
      setApplication(updated);
    } finally {
      setSavingStatus(false);
    }
  }

  if (loadError) {
    return (
      <AppLayout>
        <p style={{ color: "crimson" }}>{loadError}</p>
      </AppLayout>
    );
  }
  if (loading || !application) {
    return (
      <AppLayout>
        <p>Carregando...</p>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <h1>{application.name}</h1>
      {application.description && <p>{application.description}</p>}
      <label>
        Status:{" "}
        <select
          value={application.status}
          onChange={(e) => handleStatusChange(e.target.value as ApplicationStatus)}
          disabled={savingStatus}
        >
          <option value="RASCUNHO">Rascunho</option>
          <option value="ATIVA">Ativa</option>
          <option value="ENCERRADA">Encerrada</option>
        </select>
      </label>

      <h2 style={{ marginTop: 24 }}>Cadernos</h2>
      {booklets.length === 0 ? (
        <p style={{ color: "#666" }}>Nenhum caderno ainda.</p>
      ) : (
        <ul>
          {booklets.map((b) => (
            <li key={b.id}>
              <Link to={`/cadernos/${b.id}`}>{b.name}</Link>
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={handleCreateBooklet} style={{ display: "flex", gap: 8, marginTop: 12 }}>
        <input
          placeholder="Nome do caderno (ex.: Caderno 1)"
          value={newBookletName}
          onChange={(e) => setNewBookletName(e.target.value)}
          required
        />
        <button type="submit" disabled={creating}>
          {creating ? "Criando..." : "Adicionar caderno"}
        </button>
      </form>
      {createError && <p style={{ color: "crimson" }}>{createError}</p>}
    </AppLayout>
  );
}
