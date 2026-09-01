package subjects

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"boraif/internal/apiutil"
	"boraif/internal/auth"
	"boraif/internal/users"
)

type Handlers struct {
	Repo *Repository
}

type subjectResponse struct {
	ID           int64  `json:"id"`
	DisciplineID int64  `json:"disciplineId"`
	Name         string `json:"name"`
}

func toResponse(s Subject) subjectResponse {
	return subjectResponse{ID: s.ID, DisciplineID: s.DisciplineID, Name: s.Name}
}

type similarResponse struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	var disciplineID *int64
	if raw := r.URL.Query().Get("disciplineId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			apiutil.WriteError(w, http.StatusBadRequest, "invalid disciplineId")
			return
		}
		disciplineID = &id
	}

	list, err := h.Repo.List(r.Context(), disciplineID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list subjects")
		return
	}
	response := make([]subjectResponse, 0, len(list))
	for _, s := range list {
		response = append(response, toResponse(s))
	}
	apiutil.WriteJSON(w, http.StatusOK, response)
}

type createRequest struct {
	DisciplineID     int64  `json:"disciplineId"`
	Name             string `json:"name"`
	ConfirmDuplicate bool   `json:"confirmDuplicate"`
}

// Create implementa a seção 14: ADMIN pode criar assunto para qualquer
// disciplina; ELABORADOR só para a própria disciplina; GESTOR não cria
// assuntos. Antes de gravar, avisa sobre nomes iguais/parecidos já
// existentes na mesma disciplina, a menos que o cliente confirme a criação
// mesmo assim (confirmDuplicate).
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := auth.CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "name é obrigatório")
		return
	}

	switch currentUser.Role {
	case users.RoleAdmin:
		// pode criar para qualquer disciplina
	case users.RoleElaborador:
		if currentUser.DisciplineID == nil || *currentUser.DisciplineID != req.DisciplineID {
			apiutil.WriteError(w, http.StatusForbidden, "só é possível criar assuntos da própria disciplina")
			return
		}
	default:
		apiutil.WriteError(w, http.StatusForbidden, "papel sem permissão para criar assuntos")
		return
	}

	if !req.ConfirmDuplicate {
		similar, err := h.Repo.FindSimilar(r.Context(), req.DisciplineID, name)
		if err != nil {
			apiutil.WriteError(w, http.StatusInternalServerError, "could not check for similar subjects")
			return
		}
		if len(similar) > 0 {
			response := make([]similarResponse, 0, len(similar))
			for _, s := range similar {
				response = append(response, similarResponse{ID: s.ID, Name: s.Name, Score: s.Score})
			}
			apiutil.WriteJSON(w, http.StatusConflict, map[string]any{
				"error":   "possíveis assuntos parecidos já existem nesta disciplina",
				"similar": response,
			})
			return
		}
	}

	userID := currentUser.ID
	id, err := h.Repo.Create(r.Context(), Subject{
		DisciplineID: req.DisciplineID,
		Name:         name,
		CreatedBy:    &userID,
	})
	switch {
	case errors.Is(err, ErrDuplicateName):
		apiutil.WriteError(w, http.StatusConflict, "já existe um assunto com este nome nesta disciplina")
		return
	case errors.Is(err, ErrInvalidReference):
		apiutil.WriteError(w, http.StatusBadRequest, "disciplina inválida")
		return
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not create subject")
		return
	}

	apiutil.WriteJSON(w, http.StatusCreated, subjectResponse{ID: id, DisciplineID: req.DisciplineID, Name: name})
}

type updateRequest struct {
	Name string `json:"name"`
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "name é obrigatório")
		return
	}

	err = h.Repo.Update(r.Context(), id, name)
	switch {
	case errors.Is(err, ErrNotFound):
		apiutil.WriteError(w, http.StatusNotFound, "subject not found")
	case errors.Is(err, ErrDuplicateName):
		apiutil.WriteError(w, http.StatusConflict, "já existe um assunto com este nome nesta disciplina")
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not update subject")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	err = h.Repo.Delete(r.Context(), id)
	switch {
	case errors.Is(err, ErrNotFound):
		apiutil.WriteError(w, http.StatusNotFound, "subject not found")
	case errors.Is(err, ErrInUse):
		apiutil.WriteError(w, http.StatusConflict, "assunto em uso por questões existentes")
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not delete subject")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
