import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { AppLayout } from "../../components/AppLayout";
import { api, ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import type { Application } from "./types";

// Área "Aplicações" (seção 21/36): campanhas de prova, cada uma com um ou
// mais cadernos (seção 21.1, gerenciados na tela de detalhe).
export function ApplicationsListPage() {
  const { user } = useAuth();
  const [applications, setApplications] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .get<Application[]>("/api/applications")
      .then((list) => {
        if (!cancelled) setApplications(list);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setCreating(true);
    setError(null);
    try {
      const created = await api.post<Application>("/api/applications", { name, description });
      setApplications((prev) => [created, ...prev]);
      setName("");
      setDescription("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível criar a aplicação.");
    } finally {
      setCreating(false);
    }
  }

  return (
    <AppLayout>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>Aplicações</h1>
        {user?.role === "ADMIN" && <Link to="/configuracao-padrao">Configuração padrão</Link>}
      </div>

      {loading ? (
        <p>Carregando...</p>
      ) : (
        <table style={{ width: "100%", borderCollapse: "collapse", marginTop: 12 }}>
          <thead>
            <tr style={{ textAlign: "left", borderBottom: "1px solid #ccc" }}>
              <th>Nome</th>
              <th>Status</th>
              <th>Criada em</th>
            </tr>
          </thead>
          <tbody>
            {applications.map((a) => (
              <tr key={a.id} style={{ borderBottom: "1px solid #eee" }}>
                <td>
                  <Link to={`/aplicacoes/${a.id}`}>{a.name}</Link>
                </td>
                <td>{a.status}</td>
                <td>{new Date(a.createdAt).toLocaleDateString("pt-BR")}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h2 style={{ marginTop: 32 }}>Nova aplicação</h2>
      <form onSubmit={handleCreate} style={{ display: "flex", flexDirection: "column", gap: 8, maxWidth: 360 }}>
        <label>
          Nome (ex.: 2026/1)
          <input value={name} onChange={(e) => setName(e.target.value)} required style={{ width: "100%" }} />
        </label>
        <label>
          Descrição (opcional)
          <input value={description} onChange={(e) => setDescription(e.target.value)} style={{ width: "100%" }} />
        </label>
        {error && <p style={{ color: "crimson" }}>{error}</p>}
        <button type="submit" disabled={creating}>
          {creating ? "Criando..." : "Criar"}
        </button>
      </form>
    </AppLayout>
  );
}
