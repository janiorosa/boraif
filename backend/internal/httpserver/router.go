package httpserver

import (
	"net/http"

	"boraif/internal/ai"
	"boraif/internal/apiutil"
	"boraif/internal/applications"
	"boraif/internal/auth"
	"boraif/internal/booklets"
	"boraif/internal/catalogs"
	"boraif/internal/disciplines"
	"boraif/internal/images"
	"boraif/internal/pdf"
	"boraif/internal/questions"
	"boraif/internal/subjects"
	"boraif/internal/users"
)

// Deps agrupa os handlers de cada domínio. Cresce a cada fase (assuntos,
// questões, imagens, aplicações, ...) sem precisar mudar a assinatura de
// NewRouter a cada vez.
type Deps struct {
	AuthMiddleware *auth.Middleware
	Auth           *auth.Handlers
	Users          *users.Handlers
	Disciplines    *disciplines.Handlers
	Subjects       *subjects.Handlers
	Catalogs       *catalogs.Handlers
	Questions      *questions.Handlers
	Images         *images.Handlers
	AI             *ai.Handlers
	Applications   *applications.Handlers
	Booklets       *booklets.Handlers
	PDF            *pdf.Handlers
	// UploadsDir raiz servida publicamente em /uploads/ (imagens, seção 13).
	UploadsDir string
}

// NewRouter monta as rotas da API usando o roteador padrão do Go (net/http),
// que desde a 1.22 já suporta método+padrão sem precisar de dependência
// externa (seção 47: preferir menos dependências).
func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		apiutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/auth/login", deps.Auth.Login)
	mux.HandleFunc("POST /api/auth/logout", deps.Auth.Logout)
	mux.HandleFunc("GET /api/auth/me", auth.RequireAuth(deps.Auth.Me))
	// Autocadastro de professores (ELABORADOR): público, sem sessão — a
	// conta nasce pendente de aprovação (ver auth.Signup e users.Approve).
	mux.HandleFunc("POST /api/auth/signup", deps.Auth.Signup)

	// "Minha conta" — API Key da OpenAI (seção 17): cada professor cadastra
	// a própria chave; nunca é devolvida em texto puro depois de salva.
	mux.HandleFunc("PUT /api/me/openai-key", auth.RequireAuth(deps.Auth.SetOwnOpenAIKey))
	mux.HandleFunc("GET /api/me/openai-key/status", auth.RequireAuth(deps.Auth.OwnOpenAIKeyStatus))

	// Administração de usuários — somente ADMIN (seção 20).
	adminOnly := auth.RequireRole(users.RoleAdmin)
	mux.HandleFunc("GET /api/users", adminOnly(deps.Users.List))
	mux.HandleFunc("POST /api/users", adminOnly(deps.Users.Create))
	mux.HandleFunc("GET /api/users/{id}", adminOnly(deps.Users.Get))
	mux.HandleFunc("PUT /api/users/{id}", adminOnly(deps.Users.Update))
	mux.HandleFunc("POST /api/users/{id}/approve", adminOnly(deps.Users.Approve))
	mux.HandleFunc("POST /api/users/{id}/reject", adminOnly(deps.Users.Reject))

	// Disciplinas são fixas (seed) e só listadas — sem dado sensível, então a
	// leitura é pública (a tela de autocadastro precisa dela sem sessão).
	mux.HandleFunc("GET /api/disciplines", deps.Disciplines.List)

	// Assuntos (seção 14): leitura é livre para qualquer autenticado; a
	// criação valida dentro do próprio handler se o papel/disciplina tem
	// permissão (ADMIN em qualquer disciplina, ELABORADOR só na própria),
	// pois essa regra depende dos dados da requisição, não só do papel.
	// Editar/excluir um assunto (que é compartilhado entre professores)
	// fica restrito a ADMIN.
	mux.HandleFunc("GET /api/subjects", auth.RequireAuth(deps.Subjects.List))
	mux.HandleFunc("POST /api/subjects", auth.RequireAuth(deps.Subjects.Create))
	mux.HandleFunc("PUT /api/subjects/{id}", adminOnly(deps.Subjects.Update))
	mux.HandleFunc("DELETE /api/subjects/{id}", adminOnly(deps.Subjects.Delete))

	// Catálogos fixos (ano, dificuldade, status) — leitura livre para popular
	// seletores no editor de questões.
	mux.HandleFunc("GET /api/grade-years", auth.RequireAuth(deps.Catalogs.ListGradeYears))
	mux.HandleFunc("GET /api/difficulties", auth.RequireAuth(deps.Catalogs.ListDifficulties))
	mux.HandleFunc("GET /api/question-statuses", auth.RequireAuth(deps.Catalogs.ListQuestionStatuses))

	// Questões (seções 6/7/15/36/37): CRUD estrutural. A autorização por
	// disciplina (ADMIN: qualquer uma; ELABORADOR: só a própria) depende dos
	// dados de cada questão, então é resolvida dentro do handler, não pelo
	// roteador — RequireAuth aqui só garante que existe uma sessão válida.
	mux.HandleFunc("GET /api/questions", auth.RequireAuth(deps.Questions.List))
	mux.HandleFunc("POST /api/questions", auth.RequireAuth(deps.Questions.Create))
	mux.HandleFunc("GET /api/questions/{id}", auth.RequireAuth(deps.Questions.Get))
	mux.HandleFunc("PUT /api/questions/{id}", auth.RequireAuth(deps.Questions.Update))
	mux.HandleFunc("DELETE /api/questions/{id}", auth.RequireAuth(deps.Questions.Delete))

	// Assistente de IA (seção 16): mesma regra de disciplina de questões,
	// resolvida dentro do handler.
	mux.HandleFunc("POST /api/questions/{id}/ai/review", auth.RequireAuth(deps.AI.Review))

	// Imagens (seção 13/36): upload e biblioteca de busca/reuso por
	// disciplina. Arquivos enviados ficam publicamente acessíveis por nome
	// aleatório e imprevisível em /uploads/ (sem autorização complexa —
	// seção 13 pede exatamente isso — e sem exigir cookie para o Chromium
	// conseguir carregar as imagens ao gerar PDF na Fase 10).
	mux.HandleFunc("POST /api/images", auth.RequireAuth(deps.Images.Upload))
	mux.HandleFunc("GET /api/images", auth.RequireAuth(deps.Images.List))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(deps.UploadsDir))))

	// Aplicações e cadernos (seções 20/21/21.1/22/23/24/27): restrito a
	// ADMIN e GESTOR — ELABORADOR não participa da geração de provas.
	appOrGestor := auth.RequireRole(users.RoleAdmin, users.RoleGestor)
	mux.HandleFunc("GET /api/applications", appOrGestor(deps.Applications.List))
	mux.HandleFunc("POST /api/applications", appOrGestor(deps.Applications.Create))
	mux.HandleFunc("GET /api/applications/{id}", appOrGestor(deps.Applications.Get))
	mux.HandleFunc("PUT /api/applications/{id}", appOrGestor(deps.Applications.Update))
	mux.HandleFunc("GET /api/applications/{id}/booklets", appOrGestor(deps.Booklets.ListForApplication))
	mux.HandleFunc("POST /api/applications/{id}/booklets", appOrGestor(deps.Booklets.Create))
	mux.HandleFunc("GET /api/booklets/{id}", appOrGestor(deps.Booklets.Get))
	mux.HandleFunc("GET /api/booklets/{id}/configuration", appOrGestor(deps.Booklets.GetConfiguration))
	mux.HandleFunc("PUT /api/booklets/{id}/configuration", appOrGestor(deps.Booklets.UpdateConfiguration))
	mux.HandleFunc("GET /api/booklets/{id}/availability", appOrGestor(deps.Booklets.Availability))
	// Configuração padrão (seção 22): qualquer um dos dois lê; só ADMIN edita
	// o modelo global copiado para os próximos cadernos.
	mux.HandleFunc("GET /api/default-configuration", appOrGestor(deps.Booklets.GetDefaultConfiguration))
	mux.HandleFunc("PUT /api/default-configuration", adminOnly(deps.Booklets.SetDefaultConfiguration))

	// Geração de PDF (seções 26-30): dispara em background e nunca bloqueia
	// a requisição (seção 2.2/30). O PDF em si exige sessão para baixar —
	// diferente de imagens, uma prova é conteúdo sensível antes da aplicação.
	mux.HandleFunc("POST /api/booklets/{id}/generate", appOrGestor(deps.PDF.Generate))
	mux.HandleFunc("GET /api/booklets/{id}/generated-documents", appOrGestor(deps.PDF.ListForBooklet))
	mux.HandleFunc("GET /api/generated-documents/{id}/file", appOrGestor(deps.PDF.DownloadFile))

	return deps.AuthMiddleware.WithUser(mux)
}
