package catalogs

import (
	"net/http"

	"boraif/internal/apiutil"
)

type Handlers struct {
	Repo *Repository
}

func (h *Handlers) ListGradeYears(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.ListGradeYears(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list grade years")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, list)
}

func (h *Handlers) ListDifficulties(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.ListDifficulties(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list difficulties")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, list)
}

func (h *Handlers) ListQuestionStatuses(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.ListQuestionStatuses(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list question statuses")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, list)
}
