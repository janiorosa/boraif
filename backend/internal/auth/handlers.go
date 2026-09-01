package auth

import (
	"errors"
	"net/http"
	"strings"

	"boraif/internal/apiutil"
	"boraif/internal/security"
	"boraif/internal/users"
)

type Handlers struct {
	Users        *users.Repository
	Sessions     *SessionStore
	CookieName   string
	CookieSecure bool
	// APIKeyEncryptionKey cifra a API Key da OpenAI de cada professor
	// (seção 17) nos endpoints "minha conta" abaixo.
	APIKeyEncryptionKey []byte
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	DisciplineID *int64 `json:"disciplineId"`
}

func toUserResponse(u users.User) userResponse {
	return userResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Role:         string(u.Role),
		DisciplineID: u.DisciplineID,
	}
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.Users.FindByEmail(r.Context(), req.Email)
	if errors.Is(err, users.ErrNotFound) {
		apiutil.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !security.CheckPassword(user.PasswordHash, req.Password) {
		apiutil.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Só reveladas depois da senha já ter sido confirmada correta — não
	// vaza se um e-mail existe para quem não sabe a senha (a mensagem
	// genérica acima cobre esse caso).
	if user.PendingApproval {
		apiutil.WriteError(w, http.StatusForbidden, "seu cadastro ainda está aguardando aprovação de um administrador")
		return
	}
	if !user.Active {
		apiutil.WriteError(w, http.StatusForbidden, "sua conta está desativada; fale com um administrador")
		return
	}

	token, expiresAt, err := h.Sessions.Create(r.Context(), user.ID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	apiutil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(h.CookieName); err == nil {
		_ = h.Sessions.Revoke(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

type setAPIKeyRequest struct {
	APIKey string `json:"apiKey"`
}

// SetOwnOpenAIKey implementa a seção 17: cada professor cadastra a própria
// chave (nunca uma chave global), cifrada com AES-256-GCM antes de gravar.
// Só ELABORADOR usa o assistente de IA (seção 20), então só ele cadastra
// chave.
func (h *Handlers) SetOwnOpenAIKey(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if currentUser.Role != users.RoleElaborador {
		apiutil.WriteError(w, http.StatusForbidden, "somente elaboradores configuram API Key da OpenAI")
		return
	}

	var req setAPIKeyRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	key := strings.TrimSpace(req.APIKey)
	if key == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "apiKey é obrigatória")
		return
	}

	ciphertext, nonce, err := security.EncryptAPIKey(key, h.APIKeyEncryptionKey)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not encrypt API key")
		return
	}
	if err := h.Users.SetAPIKey(r.Context(), currentUser.ID, ciphertext, nonce); err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not save API key")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// OwnOpenAIKeyStatus só informa se uma chave está configurada — a chave em
// si nunca é devolvida ao professor, cifrada ou não (seção 17).
func (h *Handlers) OwnOpenAIKeyStatus(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	has, err := h.Users.HasAPIKey(r.Context(), currentUser.ID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not check API key status")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, map[string]bool{"configured": has})
}

type signupRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	DisciplineID int64  `json:"disciplineId"`
}

// Signup é o autocadastro de professores (requisito acrescentado depois da
// especificação original): qualquer pessoa pode se cadastrar como
// ELABORADOR — nunca como ADMIN/GESTOR, papel nem é aceito na requisição —
// mas a conta nasce inativa e marcada como pendente. Só um ADMIN aprovando
// (POST /api/users/{id}/approve) libera o login; até lá, tentar entrar
// devolve uma mensagem específica de "aguardando aprovação" (seção Login
// acima).
func (h *Handlers) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)
	if name == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "name é obrigatório")
		return
	}
	if email == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "email é obrigatório")
		return
	}
	if len(req.Password) < 8 {
		apiutil.WriteError(w, http.StatusBadRequest, "password deve ter ao menos 8 caracteres")
		return
	}
	if req.DisciplineID == 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "disciplineId é obrigatório")
		return
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not hash password")
		return
	}

	disciplineID := req.DisciplineID
	_, err = h.Users.Create(r.Context(), users.User{
		Name:            name,
		Email:           email,
		PasswordHash:    hash,
		Role:            users.RoleElaborador,
		DisciplineID:    &disciplineID,
		Active:          false,
		PendingApproval: true,
	})
	switch {
	case errors.Is(err, users.ErrEmailTaken):
		apiutil.WriteError(w, http.StatusConflict, "já existe um cadastro com este e-mail")
		return
	case errors.Is(err, users.ErrInvalidReference):
		apiutil.WriteError(w, http.StatusBadRequest, "disciplina inválida")
		return
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not create signup")
		return
	}

	apiutil.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Cadastro enviado. Aguarde a aprovação de um administrador antes de entrar.",
	})
}
