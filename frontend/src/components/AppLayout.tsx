import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

// Cabeçalho e navegação compartilhados por todas as páginas autenticadas.
export function AppLayout({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth();

  return (
    <div>
      <header
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          padding: "12px 24px",
          borderBottom: "1px solid #ddd",
        }}
      >
        <nav style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <Link to="/" style={{ fontWeight: "bold", textDecoration: "none" }}>
            BoraIF
          </Link>
          {(user?.role === "ADMIN" || user?.role === "ELABORADOR") && <Link to="/questoes">Questões</Link>}
          {(user?.role === "ADMIN" || user?.role === "ELABORADOR") && <Link to="/imagens">Imagens</Link>}
          {(user?.role === "ADMIN" || user?.role === "ELABORADOR") && <Link to="/assuntos">Assuntos</Link>}
          {(user?.role === "ADMIN" || user?.role === "GESTOR") && <Link to="/aplicacoes">Aplicações</Link>}
          {user?.role === "ELABORADOR" && <Link to="/minha-conta">Minha Conta</Link>}
          {user?.role === "ADMIN" && <Link to="/admin/usuarios">Usuários</Link>}
        </nav>
        <div>
          <span style={{ marginRight: 12 }}>
            {user?.name} · {user?.role}
          </span>
          <button onClick={() => logout()}>Sair</button>
        </div>
      </header>
      <main style={{ padding: 24 }}>{children}</main>
    </div>
  );
}
