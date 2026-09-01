package pdf

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"boraif/internal/apiutil"
	"boraif/internal/auth"
	"boraif/internal/booklets"
	"boraif/internal/users"
)

type Handlers struct {
	Service  *Service
	Booklets *booklets.Repository
	Storage  *Storage
}

type documentResponse struct {
	ID           int64      `json:"id"`
	BookletID    int64      `json:"bookletId"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

func toDocumentResponse(d GeneratedDocument) documentResponse {
	resp := documentResponse{ID: d.ID, BookletID: d.BookletID, Status: d.Status, CreatedAt: d.CreatedAt, CompletedAt: d.CompletedAt}
	if d.ErrorMessage != nil {
		resp.ErrorMessage = *d.ErrorMessage
	}
	return resp
}

// requireAppOrGestor casa com a mesma regra de aplicações/cadernos (seção
// 20): ELABORADOR não participa da geração de provas.
func requireAppOrGestor(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	currentUser, ok := auth.CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return users.User{}, false
	}
	if currentUser.Role != users.RoleAdmin && currentUser.Role != users.RoleGestor {
		apiutil.WriteError(w, http.StatusForbidden, "papel sem permissão para gerar provas")
		return users.User{}, false
	}
	return currentUser, true
}

// Generate implementa a seção 30: valida disponibilidade antes de aceitar
// (seção 24 — nunca inicia uma geração destinada a falhar, só quando a
// configuração ainda não está congelada), cria o registro PENDING e
// responde na hora; o trabalho roda numa goroutine com contexto próprio,
// já que o contexto da requisição HTTP morre assim que a resposta é enviada.
func (h *Handlers) Generate(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := requireAppOrGestor(w, r)
	if !ok {
		return
	}
	bookletID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	isFrozen, _, err := h.Service.Repo.IsFrozen(r.Context(), bookletID)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "booklet not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not check booklet configuration")
		return
	}

	if !isFrozen {
		items, err := h.Booklets.ValidateAvailability(r.Context(), bookletID)
		if err != nil {
			apiutil.WriteError(w, http.StatusInternalServerError, "could not validate availability")
			return
		}
		for _, it := range items {
			if !it.Sufficient() {
				apiutil.WriteError(w, http.StatusConflict,
					"questões insuficientes para uma ou mais cotas — verifique a disponibilidade antes de gerar")
				return
			}
		}
	}

	documentID, err := h.Service.Generate(r.Context(), bookletID, currentUser.ID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not start generation")
		return
	}

	go h.Service.Process(context.Background(), documentID, bookletID)

	apiutil.WriteJSON(w, http.StatusAccepted, map[string]any{"id": documentID, "status": StatusPending})
}

func (h *Handlers) ListForBooklet(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAppOrGestor(w, r); !ok {
		return
	}
	bookletID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	list, err := h.Service.Repo.ListDocumentsForBooklet(r.Context(), bookletID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list generated documents")
		return
	}
	result := make([]documentResponse, 0, len(list))
	for _, d := range list {
		result = append(result, toDocumentResponse(d))
	}
	apiutil.WriteJSON(w, http.StatusOK, result)
}

// DownloadFile serve o PDF já gerado. Diferente das imagens (seção 13, sem
// autorização), uma prova é conteúdo sensível antes da aplicação de fato —
// por isso exige sessão com papel ADMIN/GESTOR, e só quando COMPLETED.
func (h *Handlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAppOrGestor(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	doc, err := h.Service.Repo.FindDocumentByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load document")
		return
	}
	if doc.Status != StatusCompleted || doc.FilePath == nil {
		apiutil.WriteError(w, http.StatusConflict, "documento ainda não está pronto")
		return
	}

	file, err := os.Open(h.Storage.FullPath(*doc.FilePath))
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not open pdf file")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="caderno-`+strconv.FormatInt(doc.BookletID, 10)+`.pdf"`)
	http.ServeContent(w, r, "", doc.CreatedAt, file)
}
