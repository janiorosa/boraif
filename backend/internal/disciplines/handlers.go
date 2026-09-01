package disciplines

import (
	"net/http"

	"boraif/internal/apiutil"
)

type Handlers struct {
	Repo *Repository
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.List(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list disciplines")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, list)
}
