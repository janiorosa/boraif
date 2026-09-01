package applications

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"boraif/internal/apiutil"
	"boraif/internal/auth"
)

type Handlers struct {
	Repo *Repository
}

type response struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatorID   int64     `json:"creatorId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func toResponse(a Application) response {
	return response{
		ID: a.ID, Name: a.Name, Description: a.Description, Status: a.Status,
		CreatorID: a.CreatorID, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.List(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list applications")
		return
	}
	result := make([]response, 0, len(list))
	for _, a := range list {
		result = append(result, toResponse(a))
	}
	apiutil.WriteJSON(w, http.StatusOK, result)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	app, err := h.Repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load application")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toResponse(app))
}

type upsertRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Create implementa a seção 21: ADMIN e GESTOR podem criar aplicações
// (seção 20). O status inicial é sempre RASCUNHO.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := auth.CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req upsertRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "name é obrigatório")
		return
	}

	id, err := h.Repo.Create(r.Context(), name, strings.TrimSpace(req.Description), currentUser.ID)
	if errors.Is(err, ErrDuplicateName) {
		apiutil.WriteError(w, http.StatusConflict, "já existe uma aplicação com este nome")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not create application")
		return
	}

	created, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "application created but could not be loaded")
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

var validStatuses = map[string]bool{StatusRascunho: true, StatusAtiva: true, StatusEncerrada: true}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req upsertRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		apiutil.WriteError(w, http.StatusBadRequest, "name é obrigatório")
		return
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if !validStatuses[status] {
		apiutil.WriteError(w, http.StatusBadRequest, "status deve ser RASCUNHO, ATIVA ou ENCERRADA")
		return
	}

	err = h.Repo.Update(r.Context(), id, name, strings.TrimSpace(req.Description), status)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "application not found")
		return
	}
	if errors.Is(err, ErrDuplicateName) {
		apiutil.WriteError(w, http.StatusConflict, "já existe uma aplicação com este nome")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not update application")
		return
	}

	updated, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load updated application")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toResponse(updated))
}
