package ai

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"boraif/internal/apiutil"
	"boraif/internal/auth"
	"boraif/internal/questions"
	"boraif/internal/security"
	"boraif/internal/users"
)

type Handlers struct {
	Questions           *questions.Repository
	Users               *users.Repository
	Client              *Client
	APIKeyEncryptionKey []byte
}

type alternativeDTO struct {
	Position  string `json:"position"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

type reviewRequest struct {
	Target       string           `json:"target"`
	GradeYear    string           `json:"gradeYear"`
	Difficulty   string           `json:"difficulty"`
	Statement    string           `json:"statement"`
	Command      string           `json:"command"`
	Alternatives []alternativeDTO `json:"alternatives"`
}

var validTargets = map[string]bool{"statement": true, "command": true, "alternatives": true, "full": true}

// Review implementa POST /api/questions/{id}/ai/review (seção 16/34): o
// professor pede uma análise de um dos quatro alvos, usando sua própria
// API Key (seção 17), e recebe críticas/sugestões — nunca uma reescrita
// automática do conteúdo.
func (h *Handlers) Review(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := auth.CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if currentUser.Role != users.RoleAdmin && currentUser.Role != users.RoleElaborador {
		apiutil.WriteError(w, http.StatusForbidden, "papel sem permissão para usar o assistente de IA")
		return
	}

	questionID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	q, _, err := h.Questions.FindByID(r.Context(), questionID)
	if errors.Is(err, questions.ErrNotFound) {
		apiutil.WriteError(w, http.StatusNotFound, "question not found")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load question")
		return
	}
	if currentUser.Role == users.RoleElaborador &&
		(currentUser.DisciplineID == nil || *currentUser.DisciplineID != q.DisciplineID) {
		apiutil.WriteError(w, http.StatusForbidden, "questão de outra disciplina")
		return
	}

	var req reviewRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validTargets[req.Target] {
		apiutil.WriteError(w, http.StatusBadRequest, "target deve ser statement, command, alternatives ou full")
		return
	}

	ciphertext, nonce, err := h.Users.APIKeyFor(r.Context(), currentUser.ID)
	if errors.Is(err, users.ErrAPIKeyNotConfigured) {
		apiutil.WriteError(w, http.StatusBadRequest, "configure sua API Key da OpenAI antes de usar o assistente de IA")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not load API key")
		return
	}

	apiKey, err := security.DecryptAPIKey(ciphertext, nonce, h.APIKeyEncryptionKey)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not decrypt API key")
		return
	}

	alternatives := make([]AlternativeInput, 0, len(req.Alternatives))
	for _, a := range req.Alternatives {
		alternatives = append(alternatives, AlternativeInput{Position: a.Position, Text: a.Text, IsCorrect: a.IsCorrect})
	}

	systemPrompt := systemPromptFor(req.Target)
	userPrompt := buildUserPrompt(req.GradeYear, req.Difficulty, req.Statement, req.Command, alternatives)

	result, err := h.Client.Review(r.Context(), apiKey, systemPrompt, userPrompt)
	switch {
	case errors.Is(err, ErrUnauthorized):
		apiutil.WriteError(w, http.StatusBadRequest, "sua API Key da OpenAI foi rejeitada; verifique se ela está correta")
		return
	case err != nil:
		// A mensagem ao professor fica genérica de propósito (não é lugar
		// de vazar detalhe técnico), mas o motivo real precisa aparecer no
		// log do backend — sem isso, um erro aqui é impossível de
		// diagnosticar de fora (foi exatamente o que faltou até agora).
		log.Printf("ai review failed (question=%d, target=%s): %v", questionID, req.Target, err)
		apiutil.WriteError(w, http.StatusBadGateway, "não foi possível obter a análise da IA agora; tente novamente")
		return
	}

	apiutil.WriteJSON(w, http.StatusOK, result)
}
