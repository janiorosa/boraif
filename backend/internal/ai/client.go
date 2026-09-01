package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.openai.com/v1/chat/completions"

// ErrUnauthorized indica que a OpenAI rejeitou a API Key do professor — o
// handler traduz isso numa mensagem pedindo para conferir a chave
// cadastrada, sem nunca logar ou expor a chave em si.
var ErrUnauthorized = errors.New("openai rejected the api key")

// Client fala com a API de Chat Completions da OpenAI usando só a stdlib
// (seção 47 — menos dependências); não há necessidade de um SDK inteiro
// para um único tipo de chamada.
type Client struct {
	Model   string
	BaseURL string
	HTTP    *http.Client
}

func NewClient(model string) *Client {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &Client{
		Model:   model,
		BaseURL: defaultBaseURL,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Review envia o par de prompts para a OpenAI e decodifica a resposta como
// ReviewResult. apiKey já deve estar descriptografada pelo chamador.
func (c *Client) Review(ctx context.Context, apiKey, systemPrompt, userPrompt string) (ReviewResult, error) {
	reqBody := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
	}
	reqBody.ResponseFormat.Type = "json_object"

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return ReviewResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(payload))
	if err != nil {
		return ReviewResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("calling OpenAI: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("reading OpenAI response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return ReviewResult{}, ErrUnauthorized
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ReviewResult{}, fmt.Errorf("decoding OpenAI response: %w", err)
	}
	if parsed.Error != nil {
		return ReviewResult{}, fmt.Errorf("OpenAI error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return ReviewResult{}, errors.New("empty response from OpenAI")
	}

	var result ReviewResult
	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &result); err != nil {
		return ReviewResult{}, fmt.Errorf("decoding review JSON: %w", err)
	}
	return result, nil
}
