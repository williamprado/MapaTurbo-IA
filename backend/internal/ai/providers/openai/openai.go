package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mapaturbo-ia/internal/ai/domain"
)

type Provider struct {
	apiKey  string
	baseURL string
}

func NewProvider(apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

func (p *Provider) TestConnection(ctx context.Context) (bool, string, error) {
	if p.apiKey == "" {
		return false, "API Key is missing", nil
	}

	url := fmt.Sprintf("%s/models", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "Failed to connect to OpenAI endpoint: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false, "Invalid API Key (Unauthorized)", nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Sprintf("OpenAI API returned status %d: %s", resp.StatusCode, string(bodyBytes)), nil
	}

	return true, "Connection successful: models retrieved.", nil
}

func (p *Provider) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	return "", errors.New("generative completion not active in Phase 3A")
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionsRequest struct {
	Model          string         `json:"model"`
	Messages       []ChatMessage  `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
	Temperature    float64        `json:"temperature"`
}

type ChatCompletionsResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func (p *Provider) GenerateMindMap(ctx context.Context, input domain.GenerateMindMapInput) (*domain.MindMapAIResult, error) {
	if p.apiKey == "" {
		return nil, errors.New("OpenAI API key is missing")
	}

	// 1. Build prompt
	systemPrompt := `Você é um especialista em transformar conteúdos em mapas mentais claros, didáticos e altamente estruturados.
Gere um mapa mental seguindo exatamente o schema JSON solicitado.
Regras de Schema:
- Responda apenas com um objeto JSON válido.
- Não use Markdown (não use blocos de código com ` + "```" + `json).
- Não adicione explicações ou textos fora do JSON.
- O objeto deve conter as chaves: "title" (string), "centralTopic" (string), "summary" (string), "nodes" (array de objetos) e "edges" (array de objetos).
- Cada item em "nodes" precisa ter: "id" (string), "parentId" (string ou null), "title" (string), "content" (string), "level" (integer) e "order" (integer).
- Deve existir exatamente um node "root" com "parentId" igual a null e "level" igual a 0.
- Todo node que não for o "root" deve conter o "parentId" de seu node pai direto.
- Cada item em "edges" precisa ter: "source" (string - ID de origem) e "target" (string - ID de destino).
- Use títulos curtos e explicações objetivas para os nodes.`

	var userPrompt string
	if input.Type == "TOPIC" {
		userPrompt = fmt.Sprintf("Gere um mapa mental sobre o tema: \"%s\". Detalhes adicionais do tema: \"%s\". Profundidade máxima de ramos: %d. Idioma: %s. Estilo de aprendizado: %s.",
			input.Title, input.Content, input.Depth, input.Language, input.Style)
	} else {
		userPrompt = fmt.Sprintf("Gere um mapa mental resumindo o seguinte conteúdo de texto: \"%s\". Título do mapa: \"%s\". Profundidade máxima de ramos: %d. Idioma: %s. Estilo de aprendizado: %s.",
			input.Content, input.Title, input.Depth, input.Language, input.Style)
	}

	// 2. Call OpenAI Chat Completions with JSON Mode
	reqBody := ChatCompletionsRequest{
		Model: "gpt-4o", // Default model, handler can override if custom models table exists, but we use this.
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
		Temperature:    0.2,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: status=%d body=%s", resp.StatusCode, string(respBytes))
	}

	var chatResp ChatCompletionsResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI API response: %w", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return nil, errors.New("OpenAI returned an empty choice response")
	}

	rawJSON := chatResp.Choices[0].Message.Content

	// Clean Markdown if returned
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	// 3. Parse JSON Result
	var result domain.MindMapAIResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI mind map JSON: %w", err)
	}

	result.RawPayload = rawJSON
	return &result, nil
}

func (p *Provider) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if p.apiKey == "" {
		return nil, errors.New("OpenAI API Key is missing")
	}

	url := fmt.Sprintf("%s/embeddings", p.baseURL)
	payload := map[string]interface{}{
		"input": text,
		"model": "text-embedding-3-small",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI embedding API error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, errors.New("no embedding data returned from OpenAI")
	}

	return response.Data[0].Embedding, nil
}

