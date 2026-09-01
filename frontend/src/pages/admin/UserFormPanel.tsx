import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError } from "../../api/client";
import type { AdminUser, Discipline, Role } from "./types";

interface Props {
  disciplines: Discipline[];
  editingUser: AdminUser | null;
  onSaved: (user: AdminUser) => void;
  onCancel: () => void;
}

const ROLES: Role[] = ["ADMIN", "ELABORADOR", "GESTOR"];

export function UserFormPanel({ disciplines, editingUser, onSaved, onCancel }: Props) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<Role>("ELABORADOR");
  const [disciplineId, setDisciplineId] = useState<number | "">("");
  const [active, setActive] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    setError(null);
    setPassword("");
    if (editingUser) {
      setName(editingUser.name);
      setEmail(editingUser.email);
      setRole(editingUser.role);
      setDisciplineId(editingUser.disciplineId ?? "");
      setActive(editingUser.active);
    } else {
      setName("");
      setEmail("");
      setRole("ELABORADOR");
      setDisciplineId("");
      setActive(true);
    }
  }, [editingUser]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    const payload = {
      name,
      email,
      password,
      role,
      disciplineId: role === "ELABORADOR" ? Number(disciplineId) : null,
      active,
    };

    try {
      const saved = editingUser
        ? await api.put<AdminUser>(`/api/users/${editingUser.id}`, payload)
        : await api.post<AdminUser>("/api/users", payload);
      onSaved(saved);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível salvar o usuário.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 10, maxWidth: 360 }}>
      <h2 style={{ margin: 0 }}>{editingUser ? `Editar: ${editingUser.name}` : "Novo usuário"}</h2>

      <label>
        Nome
        <input value={name} onChange={(e) => setName(e.target.value)} required style={{ width: "100%" }} />
      </label>

      <label>
        E-mail
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          style={{ width: "100%" }}
        />
      </label>

      <label>
        {editingUser ? "Nova senha (deixe em branco para manter)" : "Senha"}
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required={!editingUser}
          minLength={editingUser && password === "" ? undefined : 8}
          style={{ width: "100%" }}
        />
      </label>

      <label>
        Papel
        <select value={role} onChange={(e) => setRole(e.target.value as Role)} style={{ width: "100%" }}>
          {ROLES.map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </select>
      </label>

      {role === "ELABORADOR" && (
        <label>
          Disciplina
          <select
            value={disciplineId}
            onChange={(e) => setDisciplineId(Number(e.target.value))}
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
        <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} /> Ativo
      </label>

      {error && <p style={{ color: "crimson", margin: 0 }}>{error}</p>}

      <div style={{ display: "flex", gap: 8 }}>
        <button type="submit" disabled={submitting}>
          {submitting ? "Salvando..." : "Salvar"}
        </button>
        <button type="button" onClick={onCancel} disabled={submitting}>
          Cancelar
        </button>
      </div>
    </form>
  );
}
