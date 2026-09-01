package auth

import (
	"context"
	"net/http"

	"boraif/internal/users"
)

type contextKey int

const userContextKey contextKey = iota

// Middleware resolve a sessão a partir do cookie e injeta o usuário autenticado
// no contexto da requisição. Requisições sem sessão válida seguem sem usuário
// no contexto; rotas que exigem autenticação usam RequireAuth.
type Middleware struct {
	Sessions   *SessionStore
	Users      *users.Repository
	CookieName string
}

func (m *Middleware) WithUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(m.CookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		userID, err := m.Sessions.UserIDForToken(r.Context(), cookie.Value)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.Users.FindByID(r.Context(), userID)
		if err != nil || !user.Active {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth bloqueia a requisição com 401 caso nenhum usuário tenha sido
// resolvido pelo WithUser.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := CurrentUser(r.Context()); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// RequireRole bloqueia a requisição com 403 caso o usuário autenticado não
// possua um dos papéis informados. A autorização é sempre validada aqui no
// backend — nunca apenas escondendo botões no frontend (seção 20/35).
func RequireRole(roles ...users.Role) func(http.HandlerFunc) http.HandlerFunc {
	allowed := make(map[users.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !allowed[user.Role] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
}

func CurrentUser(ctx context.Context) (users.User, bool) {
	u, ok := ctx.Value(userContextKey).(users.User)
	return u, ok
}
