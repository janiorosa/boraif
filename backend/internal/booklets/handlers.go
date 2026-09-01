package booklets

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"boraif/internal/apiutil"
)

type Handlers struct {
	Repo *Repository
}

type response struct {
	ID            int64  `json:"id"`
	ApplicationID int64  `json:"applicationId"`
	Name          string `json:"name"`
	SortOrder     int16  `json:"sortOrder"`
}

func toResponse(b Booklet) response {
	return response{ID: b.ID, ApplicationID: b.ApplicationID, Name: b.Name, SortOrder: b.SortOrder}
}

type quotaRuleDTO struct {
	ID           int64  `json:"id,omitempty"`
	DisciplineID int64  `json:"disciplineId"`
	SubjectID    *int64 `json:"subjectId,omitempty"`
	DifficultyID *int64 `json:"difficultyId,omitempty"`
	Quantity     int    `json:"quantity"`
}

type configurationResponse struct {
	BookletID      int64          `json:"bookletId,omitempty"`
	TotalQuestions int            `json:"totalQuestions"`
	IsFrozen       bool           `json:"isFrozen"`
	GradeYearIDs   []int64        `json:"gradeYearIds"`
	QuotaRules     []quotaRuleDTO `json:"quotaRules"`
}

func toConfigurationResponse(c Configuration) configurationResponse {
	rules := make([]quotaRuleDTO, 0, len(c.QuotaRules))
	for _, q := range c.QuotaRules {
		rules = append(rules, quotaRuleDTO{
			ID: q.ID, DisciplineID: q.DisciplineID, SubjectID: q.SubjectID, DifficultyID: q.DifficultyID, Quantity: q.Quantity,
		})
	}
	years := c.GradeYearIDs
	if years == nil {
		years = []int64{}
	}
	return configurationResponse{
		BookletID: c.BookletID, TotalQuestions: c.TotalQuestions, IsFrozen: c.IsFrozen,
		GradeYearIDs: years, QuotaRules: rules,
	}
}

func (h *Handlers) ListForApplication(w http.ResponseWriter, r *http.Request) {
	applicationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid application id")
		return
	}
	list, err := h.Repo.ListForApplication(r.Context(), applicationID)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list booklets")
		return
	}
	result := make([]response, 0, len(list))
	for _, b := range list {
		result = append(result, toResponse(b))
	}
	apiutil.WriteJSON(w, http.StatusOK, result)
}

type createRequest struct {
	Name string `json:"name"`
}

// Create implementa a seção 21.1: uma aplicação pode ter mais de um
// caderno; cada um nasce com a configuração padrão já copiada (seção 22).
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	applicationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid application id")
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

	id, err := h.Repo.Create(r.Context(), applicationID, name)
	switch {
	case errors.Is(err, ErrDuplicateName):
		apiutil.WriteError(w, http.StatusConflict, "já existe um caderno com este nome nesta aplicação")
		return
	case errors.Is(err, ErrInvalidReference):
		apiutil.WriteError(w, http.StatusBadRequest, "aplicação inválida")
		return
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not create booklet")
		return
	}

	booklet, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "booklet created but could not be loaded")
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, toResponse(booklet))
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	booklet, err := h.Repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "booklet not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load booklet")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toResponse(booklet))
}

func (h *Handlers) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	cfg, err := h.Repo.GetConfiguration(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "booklet not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load configuration")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toConfigurationResponse(cfg))
}

type updateConfigurationRequest struct {
	TotalQuestions int            `json:"totalQuestions"`
	GradeYearIDs   []int64        `json:"gradeYearIds"`
	QuotaRules     []quotaRuleDTO `json:"quotaRules"`
}

// UpdateConfiguration implementa as seções 22/23/27: total de questões,
// anos e cotas por disciplina/assunto/dificuldade, validando a soma das
// cotas (seção 23) e recusando alteração se o caderno já foi congelado
// (seção 27 — congelamento acontece na geração, Fase 10).
func (h *Handlers) UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateConfigurationRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TotalQuestions <= 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "totalQuestions deve ser maior que zero")
		return
	}

	quotas := make([]QuotaRule, 0, len(req.QuotaRules))
	for _, q := range req.QuotaRules {
		if q.DisciplineID == 0 || q.Quantity <= 0 {
			apiutil.WriteError(w, http.StatusBadRequest, "cada cota precisa de disciplineId e quantity maior que zero")
			return
		}
		quotas = append(quotas, QuotaRule{DisciplineID: q.DisciplineID, SubjectID: q.SubjectID, DifficultyID: q.DifficultyID, Quantity: q.Quantity})
	}

	err = h.Repo.UpdateConfiguration(r.Context(), id, req.TotalQuestions, req.GradeYearIDs, quotas)
	switch {
	case errors.Is(err, ErrNotFound):
		apiutil.WriteError(w, http.StatusNotFound, "booklet not found")
		return
	case errors.Is(err, ErrFrozen):
		apiutil.WriteError(w, http.StatusConflict, "a configuração deste caderno já foi congelada e não pode mais ser alterada")
		return
	case errors.Is(err, ErrQuotaMismatch):
		apiutil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, ErrInvalidReference):
		apiutil.WriteError(w, http.StatusBadRequest, "ano/disciplina/assunto/dificuldade inválidos")
		return
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not update configuration")
		return
	}

	cfg, err := h.Repo.GetConfiguration(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "configuration updated but could not be loaded")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toConfigurationResponse(cfg))
}

type availabilityItemResponse struct {
	DisciplineName string `json:"disciplineName"`
	SubjectName    string `json:"subjectName,omitempty"`
	DifficultyName string `json:"difficultyName,omitempty"`
	Requested      int    `json:"requested"`
	Available      int    `json:"available"`
	Sufficient     bool   `json:"sufficient"`
}

type availabilityResponse struct {
	Items []availabilityItemResponse `json:"items"`
	AllOK bool                       `json:"allOk"`
}

// Availability implementa a seção 24: antes de gerar, mostra quantas
// questões foram pedidas vs. quantas existem de fato para cada cota.
func (h *Handlers) Availability(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	items, err := h.Repo.ValidateAvailability(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "booklet not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not validate availability")
		return
	}

	response := availabilityResponse{Items: make([]availabilityItemResponse, 0, len(items)), AllOK: true}
	for _, it := range items {
		sufficient := it.Sufficient()
		if !sufficient {
			response.AllOK = false
		}
		response.Items = append(response.Items, availabilityItemResponse{
			DisciplineName: it.DisciplineName, SubjectName: it.SubjectName, DifficultyName: it.DifficultyName,
			Requested: it.Requested, Available: it.Available, Sufficient: sufficient,
		})
	}
	apiutil.WriteJSON(w, http.StatusOK, response)
}

func (h *Handlers) GetDefaultConfiguration(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.Repo.GetDefaultConfiguration(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load default configuration")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toConfigurationResponse(cfg))
}

// SetDefaultConfiguration implementa a seção 22 (ADMIN apenas): alterar o
// padrão nunca muda cadernos já criados, só o que vale para os próximos.
func (h *Handlers) SetDefaultConfiguration(w http.ResponseWriter, r *http.Request) {
	var req updateConfigurationRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TotalQuestions < 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "totalQuestions não pode ser negativo")
		return
	}

	quotas := make([]QuotaRule, 0, len(req.QuotaRules))
	sum := 0
	for _, q := range req.QuotaRules {
		if q.DisciplineID == 0 || q.Quantity <= 0 {
			apiutil.WriteError(w, http.StatusBadRequest, "cada cota precisa de disciplineId e quantity maior que zero")
			return
		}
		sum += q.Quantity
		quotas = append(quotas, QuotaRule{DisciplineID: q.DisciplineID, SubjectID: q.SubjectID, DifficultyID: q.DifficultyID, Quantity: q.Quantity})
	}
	if sum != req.TotalQuestions {
		apiutil.WriteError(w, http.StatusBadRequest, "a soma das cotas precisa ser igual a totalQuestions")
		return
	}

	if err := h.Repo.SetDefaultConfiguration(r.Context(), req.TotalQuestions, req.GradeYearIDs, quotas); err != nil {
		if errors.Is(err, ErrInvalidReference) {
			apiutil.WriteError(w, http.StatusBadRequest, "ano/disciplina/assunto/dificuldade inválidos")
			return
		}
		apiutil.WriteError(w, http.StatusInternalServerError, "could not save default configuration")
		return
	}

	cfg, err := h.Repo.GetDefaultConfiguration(r.Context())
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "configuration saved but could not be loaded")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toConfigurationResponse(cfg))
}
