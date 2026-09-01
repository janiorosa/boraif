import type { ReactElement } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth, type CurrentUser } from "./auth/AuthContext";
import { LoginPage } from "./pages/LoginPage";
import { SignupPage } from "./pages/SignupPage";
import { DashboardPage } from "./pages/DashboardPage";
import { UsersPage } from "./pages/admin/UsersPage";
import { SubjectsPage } from "./pages/subjects/SubjectsPage";
import { QuestionsListPage } from "./pages/questions/QuestionsListPage";
import { NewQuestionPage } from "./pages/questions/NewQuestionPage";
import { QuestionEditPage } from "./pages/questions/QuestionEditPage";
import { ImageLibraryPage } from "./pages/images/ImageLibraryPage";
import { MyAccountPage } from "./pages/account/MyAccountPage";
import { ApplicationsListPage } from "./pages/applications/ApplicationsListPage";
import { ApplicationDetailPage } from "./pages/applications/ApplicationDetailPage";
import { BookletConfigPage } from "./pages/applications/BookletConfigPage";
import { DefaultConfigurationPage } from "./pages/applications/DefaultConfigurationPage";

function RequireAuth({ children }: { children: ReactElement }) {
  const { user, loading } = useAuth();

  if (loading) return null;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

function RequireRole({ roles, children }: { roles: CurrentUser["role"][]; children: ReactElement }) {
  const { user, loading } = useAuth();

  if (loading) return null;
  if (!user) return <Navigate to="/login" replace />;
  if (!roles.includes(user.role)) return <Navigate to="/" replace />;
  return children;
}

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/cadastro" element={<SignupPage />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            <DashboardPage />
          </RequireAuth>
        }
      />
      <Route
        path="/admin/usuarios"
        element={
          <RequireRole roles={["ADMIN"]}>
            <UsersPage />
          </RequireRole>
        }
      />
      <Route
        path="/assuntos"
        element={
          <RequireRole roles={["ADMIN", "ELABORADOR"]}>
            <SubjectsPage />
          </RequireRole>
        }
      />
      <Route
        path="/questoes"
        element={
          <RequireRole roles={["ADMIN", "ELABORADOR"]}>
            <QuestionsListPage />
          </RequireRole>
        }
      />
      <Route
        path="/questoes/nova"
        element={
          <RequireRole roles={["ADMIN", "ELABORADOR"]}>
            <NewQuestionPage />
          </RequireRole>
        }
      />
      <Route
        path="/questoes/:id"
        element={
          <RequireRole roles={["ADMIN", "ELABORADOR"]}>
            <QuestionEditPage />
          </RequireRole>
        }
      />
      <Route
        path="/imagens"
        element={
          <RequireRole roles={["ADMIN", "ELABORADOR"]}>
            <ImageLibraryPage />
          </RequireRole>
        }
      />
      <Route
        path="/minha-conta"
        element={
          <RequireRole roles={["ELABORADOR"]}>
            <MyAccountPage />
          </RequireRole>
        }
      />
      <Route
        path="/aplicacoes"
        element={
          <RequireRole roles={["ADMIN", "GESTOR"]}>
            <ApplicationsListPage />
          </RequireRole>
        }
      />
      <Route
        path="/aplicacoes/:id"
        element={
          <RequireRole roles={["ADMIN", "GESTOR"]}>
            <ApplicationDetailPage />
          </RequireRole>
        }
      />
      <Route
        path="/cadernos/:id"
        element={
          <RequireRole roles={["ADMIN", "GESTOR"]}>
            <BookletConfigPage />
          </RequireRole>
        }
      />
      <Route
        path="/configuracao-padrao"
        element={
          <RequireRole roles={["ADMIN"]}>
            <DefaultConfigurationPage />
          </RequireRole>
        }
      />
    </Routes>
  );
}
