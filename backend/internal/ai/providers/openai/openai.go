package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
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
	return "", errors.New("generative completion not active in Phase 2B")
}
