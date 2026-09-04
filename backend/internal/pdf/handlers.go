package pdf

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
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
	ID            int64      `json:"id"`
	BookletID     int64      `json:"bookletId"`
	VariantID     int64      `json:"variantId,omitempty"`
	VariantNumber int        `json:"variantNumber,omitempty"`
	Kind          string     `json:"kind"`
	Status        string     `json:"status"`
	ErrorMessage  string     `json:"errorMessage,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}

func toDocumentResponse(d GeneratedDocument) documentResponse {
	resp := documentResponse{ID: d.ID, BookletID: d.BookletID, Kind: d.Kind, Status: d.Status, CreatedAt: d.CreatedAt, CompletedAt: d.CompletedAt}
	if d.VariantID != nil {
		resp.VariantID = *d.VariantID
	}
	if d.VariantNumber != nil {
		resp.VariantNumber = *d.VariantNumber
	}
	if d.ErrorMessage != nil {
		resp.ErrorMessage = *d.ErrorMessage
	}
	return resp
}

type variantResponse struct {
	ID            int64 `json:"id"`
	VariantNumber int   `json:"variantNumber"`
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

// Generate implementa a seção 30 + tipos de prova: valida disponibilidade
// antes de aceitar (seção 24 — nunca inicia uma geração destinada a falhar,
// só quando a configuração ainda não está congelada), congela o snapshot e
// os tipos de prova (rápido, só banco) e cria um par de registros PENDING
// (prova + gabarito) por tipo; a etapa lenta (Chromium, um documento de
// cada vez) roda numa goroutine com contexto próprio, já que o contexto da
// requisição HTTP morre assim que a resposta é enviada.
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

	documentIDs, err := h.Service.Generate(r.Context(), bookletID, currentUser.ID)
	switch {
	case errors.Is(err, ErrInsufficient):
		apiutil.WriteError(w, http.StatusConflict,
			"questões insuficientes para uma ou mais cotas — verifique a disponibilidade antes de gerar")
		return
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not start generation")
		return
	}

	go h.Service.ProcessAll(context.Background(), documentIDs)

	apiutil.WriteJSON(w, http.StatusAccepted, map[string]any{"documentIds": documentIDs, "status": StatusPending})
}

// ListVariants implementa a listagem dos "tipos de prova" já gerados para
// um caderno (vazio antes da primeira geração) — o frontend usa isso para
// saber quantas seções "Tipo N" mostrar e o id de cada variante para os
// links de download do gabarito em CSV.
func (h *Handlers) ListVariants(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAppOrGestor(w, r); !ok {
		return
	}
	bookletID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	variants, err := h.Service.Repo.ListVariants(r.Context(), bookletID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list variants")
		return
	}
	result := make([]variantResponse, 0, len(variants))
	for _, v := range variants {
		result = append(result, variantResponse{ID: v.ID, VariantNumber: v.VariantNumber})
	}
	apiutil.WriteJSON(w, http.StatusOK, result)
}

// AnswerKeyCSV serve o gabarito de um tipo de prova em CSV — gerado na hora
// a partir do que já está gravado no banco (booklet_variant_questions), sem
// depender de nenhum job de PDF ter rodado.
func (h *Handlers) AnswerKeyCSV(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAppOrGestor(w, r); !ok {
		return
	}
	variantID, err := strconv.ParseInt(r.PathValue("variantId"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid variant id")
		return
	}
	variant, err := h.Service.Repo.FindVariantByID(r.Context(), variantID)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "variant not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load variant")
		return
	}
	questions, err := h.Service.Repo.LoadVariantQuestions(r.Context(), variantID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load answer key")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="gabarito-tipo-%d.csv"`, variant.VariantNumber))

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"Questao", "Resposta"})
	for _, q := range questions {
		_ = writer.Write([]string{strconv.Itoa(q.PositionInVariant), q.CorrectLetter})
	}
	writer.Flush()
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

	name := "caderno-" + strconv.FormatInt(doc.BookletID, 10)
	if doc.VariantNumber != nil {
		name += "-tipo-" + strconv.Itoa(*doc.VariantNumber)
	}
	if doc.Kind == KindAnswerKey {
		name = "gabarito-" + name
	} else {
		name = "prova-" + name
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+name+`.pdf"`)
	http.ServeContent(w, r, "", doc.CreatedAt, file)
}
