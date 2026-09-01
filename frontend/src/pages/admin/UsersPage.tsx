import { useEffect, useState } from "react";
import { AppLayout } from "../../components/AppLayout";
import { api, ApiError } from "../../api/client";
import { UserFormPanel } from "./UserFormPanel";
import type { AdminUser, Discipline } from "./types";

// undefined = formulário escondido · null = criando novo · AdminUser = editando
type FormTarget = AdminUser | null | undefined;

export function UsersPage() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [loading, setLoading] = useState(true);
  const [formTarget, setFormTarget] = useState<FormTarget>(undefined);
  const [approvalError, setApprovalError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    Promise.all([api.get<AdminUser[]>("/api/users"), api.get<Discipline[]>("/api/disciplines")])
      .then(([userList, disciplineList]) => {
        if (cancelled) return;
        setUsers(userList);
        setDisciplines(disciplineList);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  function disciplineName(id: number | null): string {
    if (id === null) return "—";
    return disciplines.find((d) => d.id === id)?.name ?? "—";
  }

  function handleSaved(saved: AdminUser) {
    setUsers((prev) => {
      const exists = prev.some((u) => u.id === saved.id);
      return exists ? prev.map((u) => (u.id === saved.id ? saved : u)) : [...prev, saved];
    });
    setFormTarget(undefined);
  }

  async function handleApprove(pending: AdminUser) {
    setApprovalError(null);
    try {
      const approved = await api.post<AdminUser>(`/api/users/${pending.id}/approve`);
      setUsers((prev) => prev.map((u) => (u.id === approved.id ? approved : u)));
    } catch (err) {
      setApprovalError(err instanceof ApiError ? err.message : "Não foi possível aprovar o cadastro.");
    }
  }

  async function handleReject(pending: AdminUser) {
    if (!confirm(`Recusar e excluir o cadastro de "${pending.name}"?`)) return;
    setApprovalError(null);
    try {
      await api.post(`/api/users/${pending.id}/reject`);
      setUsers((prev) => prev.filter((u) => u.id !== pending.id));
    } catch (err) {
      setApprovalError(err instanceof ApiError ? err.message : "Não foi possível recusar o cadastro.");
    }
  }

  const pendingUsers = users.filter((u) => u.pendingApproval);
  const approvedUsers = users.filter((u) => !u.pendingApproval);

  return (
    <AppLayout>
      <div style={{ display: "flex", gap: 32, alignItems: "flex-start" }}>
        <div style={{ flex: 1 }}>
          {approvalError && <p style={{ color: "crimson" }}>{approvalError}</p>}

          {pendingUsers.length > 0 && (
            <div style={{ marginBottom: 32 }}>
              <h2>Cadastros pendentes de aprovação</h2>
              <p style={{ color: "#666" }}>
                Professores que se cadastraram sozinhos e ainda não podem entrar até você aprovar.
              </p>
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ textAlign: "left", borderBottom: "1px solid #ccc" }}>
                    <th>Nome</th>
                    <th>E-mail</th>
                    <th>Disciplina</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {pendingUsers.map((u) => (
                    <tr key={u.id} style={{ borderBottom: "1px solid #eee" }}>
                      <td>{u.name}</td>
                      <td>{u.email}</td>
                      <td>{disciplineName(u.disciplineId)}</td>
                      <td>
                        <button onClick={() => handleApprove(u)}>Aprovar</button>
                        <button onClick={() => handleReject(u)}>Recusar</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <h1>Usuários</h1>
            <button onClick={() => setFormTarget(null)}>Novo usuário</button>
          </div>

          {loading ? (
            <p>Carregando...</p>
          ) : (
            <table style={{ width: "100%", borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ textAlign: "left", borderBottom: "1px solid #ccc" }}>
                  <th>Nome</th>
                  <th>E-mail</th>
                  <th>Papel</th>
                  <th>Disciplina</th>
                  <th>Ativo</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {approvedUsers.map((u) => (
                  <tr key={u.id} style={{ borderBottom: "1px solid #eee" }}>
                    <td>{u.name}</td>
                    <td>{u.email}</td>
                    <td>{u.role}</td>
                    <td>{disciplineName(u.disciplineId)}</td>
                    <td>{u.active ? "Sim" : "Não"}</td>
                    <td>
                      <button onClick={() => setFormTarget(u)}>Editar</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {formTarget !== undefined && (
          <UserFormPanel
            disciplines={disciplines}
            editingUser={formTarget}
            onSaved={handleSaved}
            onCancel={() => setFormTarget(undefined)}
          />
        )}
      </div>
    </AppLayout>
  );
}
