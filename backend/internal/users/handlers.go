package users

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"boraif/internal/apiutil"
	"boraif/internal/security"
)

type Handlers struct {
	Repo *Repository
}

type userResponse struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Role            string `json:"role"`
	DisciplineID    *int64 `json:"disciplineId"`
	Active          bool   `json:"active"`
	PendingApproval bool   `json:"pendingApproval"`
}

func toResponse(u User) userResponse {
	return userResponse{
		ID:              u.ID,
		Name:            u.Name,
		Email:           u.Email,
		Role:            string(u.Role),
		DisciplineID:    u.DisciplineID,
		Active:          u.Active,
		PendingApproval: u.PendingApproval,
	}
}

type upsertRequest struct {
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	Role         string  `json:"role"`
	DisciplineID *int64  `json:"disciplineId"`
	Active       *bool   `json:"active"`
}

// validate aplica as regras da seção 15/20: papel precisa ser um dos três
// suportados, e disciplina é obrigatória para ELABORADOR e proibida para os
// demais papéis (evita dado ambíguo/inconsistente).
func (req upsertRequest) validateRoleAndDiscipline() (Role, *int64, error) {
	role := Role(strings.ToUpper(strings.TrimSpace(req.Role)))
	switch role {
	case RoleAdmin, RoleGestor:
		return role, nil, nil
	case RoleElaborador:
		if req.DisciplineID == nil {
			return "", nil, errors.New("disciplineId é obrigatório para o papel ELABORADOR")
		}
		return role, req.DisciplineID, nil
	default:
		return "", nil, errors.New("role deve ser ADMIN, ELABORADOR ou GESTOR")
	}
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.List(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	response := make([]userResponse, 0, len(list))
	for _, u := range list {
		response = append(response, toResponse(u))
	}
	apiutil.WriteJSON(w, http.StatusOK, response)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := h.Repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load user")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toResponse(u))
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req upsertRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "name é obrigatório")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "email é obrigatório")
		return
	}
	if len(req.Password) < 8 {
		apiutil.WriteError(w, http.StatusBadRequest, "password deve ter ao menos 8 caracteres")
		return
	}
	role, disciplineID, err := req.validateRoleAndDiscipline()
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not hash password")
		return
	}

	id, err := h.Repo.Create(r.Context(), User{
		Name:         strings.TrimSpace(req.Name),
		Email:        strings.TrimSpace(req.Email),
		PasswordHash: hash,
		Role:         role,
		DisciplineID: disciplineID,
		// Usuário criado por um ADMIN já nasce ativo e nunca pendente —
		// PendingApproval só existe para o autocadastro (ver auth.Signup).
		Active: true,
	})
	if errors.Is(err, ErrEmailTaken) {
		apiutil.WriteError(w, http.StatusConflict, "email já está em uso")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	created, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "user created but could not be loaded")
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update altera os dados administrativos e, opcionalmente, a senha (quando
// password não vem em branco no corpo da requisição).
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req upsertRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "name é obrigatório")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "email é obrigatório")
		return
	}
	role, disciplineID, err := req.validateRoleAndDiscipline()
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	err = h.Repo.Update(r.Context(), User{
		ID:           id,
		Name:         strings.TrimSpace(req.Name),
		Email:        strings.TrimSpace(req.Email),
		Role:         role,
		DisciplineID: disciplineID,
		Active:       active,
	})
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	if errors.Is(err, ErrEmailTaken) {
		apiutil.WriteError(w, http.StatusConflict, "email já está em uso")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not update user")
		return
	}

	if req.Password != "" {
		if len(req.Password) < 8 {
			apiutil.WriteError(w, http.StatusBadRequest, "password deve ter ao menos 8 caracteres")
			return
		}
		hash, err := security.HashPassword(req.Password)
		if err != nil {
			apiutil.WriteError(w, http.StatusInternalServerError, "could not hash password")
			return
		}
		if err := h.Repo.UpdatePassword(r.Context(), id, hash); err != nil {
			apiutil.WriteError(w, http.StatusInternalServerError, "could not update password")
			return
		}
	}

	updated, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load updated user")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// Approve libera o acesso de um autocadastro (ELABORADOR) pendente,
// marcando-o ativo. Só afeta contas realmente pendentes.
func (h *Handlers) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Repo.Approve(r.Context(), id); errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "cadastro pendente não encontrado")
		return
	} else if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not approve user")
		return
	}

	updated, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "approved but could not reload user")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// Reject descarta um autocadastro pendente (exclui a conta). Só afeta
// contas realmente pendentes — nunca serve para excluir um usuário já
// aprovado.
func (h *Handlers) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Repo.RejectPending(r.Context(), id); errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "cadastro pendente não encontrado")
		return
	} else if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not reject user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
