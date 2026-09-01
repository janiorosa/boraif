import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { api, ApiError } from "../api/client";
import type { Discipline } from "../types";

// Autocadastro de professores (requisito acrescentado depois da
// especificação original): qualquer pessoa pode se cadastrar como
// ELABORADOR — a conta nasce inativa, aguardando um ADMIN aprovar em
// /admin/usuarios. Não é possível se cadastrar como ADMIN/GESTOR por aqui.
export function SignupPage() {
  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [disciplineId, setDisciplineId] = useState<number | "">("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

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

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (disciplineId === "") return;

    setSubmitting(true);
    try {
      await api.post("/api/auth/signup", { name, email, password, disciplineId });
      setDone(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível enviar o cadastro.");
    } finally {
      setSubmitting(false);
    }
  }

  if (done) {
    return (
      <div style={{ display: "flex", minHeight: "100vh", alignItems: "center", justifyContent: "center" }}>
        <div style={{ width: 360, textAlign: "center" }}>
          <h1>Cadastro enviado</h1>
          <p>
            Sua conta foi criada e já está aguardando a aprovação de um administrador. Você receberá acesso assim
            que ela for revisada.
          </p>
          <Link to="/login">Voltar para o login</Link>
        </div>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", minHeight: "100vh", alignItems: "center", justifyContent: "center" }}>
      <form onSubmit={handleSubmit} style={{ width: 320, display: "flex", flexDirection: "column", gap: 12 }}>
        <h1 style={{ marginBottom: 8 }}>Cadastro de professor</h1>
        <p style={{ marginTop: 0, fontSize: 14, color: "#666" }}>
          Depois de enviado, sua conta precisa ser aprovada por um administrador antes de você conseguir entrar.
        </p>

        <label>
          Nome
          <input value={name} onChange={(e) => setName(e.target.value)} required autoFocus style={{ width: "100%" }} />
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
          Senha
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={8}
            style={{ width: "100%" }}
          />
        </label>
        <label>
          Disciplina
          <select
            value={disciplineId}
            onChange={(e) => setDisciplineId(e.target.value === "" ? "" : Number(e.target.value))}
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

        {error && <p style={{ color: "crimson", margin: 0 }}>{error}</p>}
        <button type="submit" disabled={submitting}>
          {submitting ? "Enviando..." : "Cadastrar"}
        </button>
        <p style={{ margin: 0, fontSize: 14 }}>
          Já tem conta? <Link to="/login">Entrar</Link>
        </p>
      </form>
    </div>
  );
}
