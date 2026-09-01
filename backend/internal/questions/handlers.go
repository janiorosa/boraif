package questions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"boraif/internal/apiutil"
	"boraif/internal/auth"
	"boraif/internal/catalogs"
	"boraif/internal/users"
)

type Handlers struct {
	Repo     *Repository
	Statuses *catalogs.Repository
}

type AlternativeDTO struct {
	Position  string          `json:"position"`
	Content   json.RawMessage `json:"content"`
	IsCorrect bool            `json:"isCorrect"`
}

type summaryResponse struct {
	ID             int64     `json:"id"`
	DisciplineName string    `json:"disciplineName"`
	SubjectName    string    `json:"subjectName"`
	GradeYearName  string    `json:"gradeYearName"`
	DifficultyName string    `json:"difficultyName"`
	StatusCode     string    `json:"statusCode"`
	StatusName     string    `json:"statusName"`
	AuthorName     string    `json:"authorName"`
	UpdatedAt      time.Time `json:"updatedAt"`
	RevisionNumber int       `json:"revisionNumber"`
}

type listResponse struct {
	Items []summaryResponse `json:"items"`
	Total int               `json:"total"`
}

type detailResponse struct {
	ID             int64            `json:"id"`
	DisciplineID   int64            `json:"disciplineId"`
	SubjectID      int64            `json:"subjectId"`
	GradeYearID    int64            `json:"gradeYearId"`
	DifficultyID   int64            `json:"difficultyId"`
	StatusID       int64            `json:"statusId"`
	AuthorID       int64            `json:"authorId"`
	Statement      json.RawMessage  `json:"statement"`
	Command        json.RawMessage  `json:"command"`
	Alternatives   []AlternativeDTO `json:"alternatives"`
	RevisionNumber int              `json:"revisionNumber"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}

func toDetailResponse(q Question, alts []Alternative) detailResponse {
	dtos := make([]AlternativeDTO, len(alts))
	for i, a := range alts {
		dtos[i] = AlternativeDTO{Position: positionNames[a.Position], Content: a.ContentJSON, IsCorrect: a.IsCorrect}
	}
	return detailResponse{
		ID:             q.ID,
		DisciplineID:   q.DisciplineID,
		SubjectID:      q.SubjectID,
		GradeYearID:    q.GradeYearID,
		DifficultyID:   q.DifficultyID,
		StatusID:       q.StatusID,
		AuthorID:       q.AuthorID,
		Statement:      q.StatementJSON,
		Command:        q.CommandJSON,
		Alternatives:   dtos,
		RevisionNumber: q.RevisionNumber,
		CreatedAt:      q.CreatedAt,
		UpdatedAt:      q.UpdatedAt,
	}
}

// authorizeRole garante que só ADMIN/ELABORADOR usam o CRUD de questões
// (GESTOR trabalha só através da geração de provas — seção 20).
func authorizeRole(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	currentUser, ok := auth.CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return users.User{}, false
	}
	if currentUser.Role != users.RoleAdmin && currentUser.Role != users.RoleElaborador {
		apiutil.WriteError(w, http.StatusForbidden, "papel sem permissão para acessar questões")
		return users.User{}, false
	}
	return currentUser, true
}

// canAccessDiscipline aplica a seção 15: ELABORADOR só acessa questões da
// própria disciplina; ADMIN acessa qualquer uma.
func canAccessDiscipline(u users.User, disciplineID int64) bool {
	if u.Role == users.RoleAdmin {
		return true
	}
	return u.DisciplineID != nil && *u.DisciplineID == disciplineID
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := authorizeRole(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	filter := ListFilter{
		Search:  query.Get("search"),
		SortBy:  query.Get("sortBy"),
		SortDir: query.Get("sortDir"),
	}
	filter.Page, _ = strconv.Atoi(query.Get("page"))
	filter.PageSize, _ = strconv.Atoi(query.Get("pageSize"))

	if id, ok, err := parseOptionalID(query, "subjectId"); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid subjectId")
		return
	} else if ok {
		filter.SubjectID = &id
	}
	if id, ok, err := parseOptionalID(query, "gradeYearId"); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid gradeYearId")
		return
	} else if ok {
		filter.GradeYearID = &id
	}
	if id, ok, err := parseOptionalID(query, "difficultyId"); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid difficultyId")
		return
	} else if ok {
		filter.DifficultyID = &id
	}
	if id, ok, err := parseOptionalID(query, "statusId"); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid statusId")
		return
	} else if ok {
		filter.StatusID = &id
	}
	if id, ok, err := parseOptionalID(query, "authorId"); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid authorId")
		return
	} else if ok {
		filter.AuthorID = &id
	}

	// seção 36: para ELABORADOR, a disciplina é sempre a própria, ignorando
	// qualquer disciplineId vindo da requisição.
	if currentUser.Role == users.RoleElaborador {
		filter.DisciplineID = currentUser.DisciplineID
	} else if id, ok, err := parseOptionalID(query, "disciplineId"); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid disciplineId")
		return
	} else if ok {
		filter.DisciplineID = &id
	}

	items, total, err := h.Repo.List(r.Context(), filter)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list questions")
		return
	}

	response := listResponse{Items: make([]summaryResponse, 0, len(items)), Total: total}
	for _, it := range items {
		response.Items = append(response.Items, summaryResponse{
			ID: it.ID, DisciplineName: it.DisciplineName, SubjectName: it.SubjectName,
			GradeYearName: it.GradeYearName, DifficultyName: it.DifficultyName,
			StatusCode: it.StatusCode, StatusName: it.StatusName, AuthorName: it.AuthorName,
			UpdatedAt: it.UpdatedAt, RevisionNumber: it.RevisionNumber,
		})
	}
	apiutil.WriteJSON(w, http.StatusOK, response)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := authorizeRole(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	q, alts, err := h.Repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "question not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load question")
		return
	}
	if !canAccessDiscipline(currentUser, q.DisciplineID) {
		apiutil.WriteError(w, http.StatusForbidden, "questão de outra disciplina")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toDetailResponse(q, alts))
}

type createRequest struct {
	DisciplineID *int64           `json:"disciplineId"`
	SubjectID    int64            `json:"subjectId"`
	GradeYearID  int64            `json:"gradeYearId"`
	DifficultyID int64            `json:"difficultyId"`
	Statement    json.RawMessage  `json:"statement"`
	Command      json.RawMessage  `json:"command"`
	Alternatives []AlternativeDTO `json:"alternatives"`
}

// Create implementa o fluxo da seção 37: o professor escolhe ano, assunto e
// dificuldade e começa a escrever; a disciplina já vem determinada pelo
// usuário (a própria, para ELABORADOR) e o status inicial é sempre RASCUNHO.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := authorizeRole(w, r)
	if !ok {
		return
	}

	var req createRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var disciplineID int64
	switch currentUser.Role {
	case users.RoleElaborador:
		if currentUser.DisciplineID == nil {
			apiutil.WriteError(w, http.StatusForbidden, "usuário sem disciplina associada")
			return
		}
		disciplineID = *currentUser.DisciplineID
		if req.DisciplineID != nil && *req.DisciplineID != disciplineID {
			apiutil.WriteError(w, http.StatusForbidden, "só é possível criar questões da própria disciplina")
			return
		}
	case users.RoleAdmin:
		if req.DisciplineID == nil {
			apiutil.WriteError(w, http.StatusBadRequest, "disciplineId é obrigatório")
			return
		}
		disciplineID = *req.DisciplineID
	}

	if req.SubjectID == 0 || req.GradeYearID == 0 || req.DifficultyID == 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "subjectId, gradeYearId e difficultyId são obrigatórios")
		return
	}
	if len(req.Statement) == 0 || len(req.Command) == 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "statement e command são obrigatórios")
		return
	}

	alts, err := ParseAlternatives(req.Alternatives)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	statusID, err := h.Statuses.StatusIDByCode(r.Context(), "RASCUNHO")
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not resolve initial status")
		return
	}

	id, err := h.Repo.Create(r.Context(), Question{
		DisciplineID:  disciplineID,
		SubjectID:     req.SubjectID,
		GradeYearID:   req.GradeYearID,
		DifficultyID:  req.DifficultyID,
		StatusID:      statusID,
		AuthorID:      currentUser.ID,
		StatementJSON: req.Statement,
		CommandJSON:   req.Command,
	}, alts)
	switch {
	case errors.Is(err, ErrInvalidReference):
		apiutil.WriteError(w, http.StatusBadRequest, "assunto/ano/dificuldade inválidos")
		return
	case errors.Is(err, ErrAlternativesInvalid):
		apiutil.WriteError(w, http.StatusBadRequest, ErrAlternativesInvalid.Error())
		return
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not create question")
		return
	}

	q, createdAlts, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "question created but could not be loaded")
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, toDetailResponse(q, createdAlts))
}

type updateRequest struct {
	SubjectID    int64            `json:"subjectId"`
	GradeYearID  int64            `json:"gradeYearId"`
	DifficultyID int64            `json:"difficultyId"`
	StatusID     int64            `json:"statusId"`
	Statement    json.RawMessage  `json:"statement"`
	Command      json.RawMessage  `json:"command"`
	Alternatives []AlternativeDTO `json:"alternatives"`
}

// Update é o mesmo endpoint que o autosave da Fase 6 vai chamar repetidamente
// com debounce: salva metadados, enunciado, comando e as cinco alternativas
// de uma vez (seção 18), nunca campo a campo.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := authorizeRole(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, _, err := h.Repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "question not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load question")
		return
	}
	if !canAccessDiscipline(currentUser, existing.DisciplineID) {
		apiutil.WriteError(w, http.StatusForbidden, "questão de outra disciplina")
		return
	}

	var req updateRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SubjectID == 0 || req.GradeYearID == 0 || req.DifficultyID == 0 || req.StatusID == 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "subjectId, gradeYearId, difficultyId e statusId são obrigatórios")
		return
	}
	if len(req.Statement) == 0 || len(req.Command) == 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "statement e command são obrigatórios")
		return
	}

	alts, err := ParseAlternatives(req.Alternatives)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.Repo.Update(r.Context(), Question{
		ID:            id,
		DisciplineID:  existing.DisciplineID,
		SubjectID:     req.SubjectID,
		GradeYearID:   req.GradeYearID,
		DifficultyID:  req.DifficultyID,
		StatusID:      req.StatusID,
		StatementJSON: req.Statement,
		CommandJSON:   req.Command,
	}, alts)
	switch {
	case errors.Is(err, ErrNotFound):
		apiutil.WriteError(w, http.StatusNotFound, "question not found")
		return
	case errors.Is(err, ErrInvalidReference):
		apiutil.WriteError(w, http.StatusBadRequest, "assunto/ano/dificuldade/status inválidos")
		return
	case errors.Is(err, ErrAlternativesInvalid):
		apiutil.WriteError(w, http.StatusBadRequest, ErrAlternativesInvalid.Error())
		return
	case err != nil:
		apiutil.WriteError(w, http.StatusInternalServerError, "could not update question")
		return
	}

	q, updatedAlts, err := h.Repo.FindByID(r.Context(), id)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "question updated but could not be loaded")
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, toDetailResponse(q, updatedAlts))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := authorizeRole(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, _, err := h.Repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "question not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load question")
		return
	}
	if !canAccessDiscipline(currentUser, existing.DisciplineID) {
		apiutil.WriteError(w, http.StatusForbidden, "questão de outra disciplina")
		return
	}

	if err := h.Repo.Delete(r.Context(), id); err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not delete question")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseOptionalID(query map[string][]string, key string) (int64, bool, error) {
	values, present := query[key]
	if !present || len(values) == 0 || values[0] == "" {
		return 0, false, nil
	}
	id, err := strconv.ParseInt(values[0], 10, 64)
	return id, true, err
}
