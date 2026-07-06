package grok

import (
	"context"
	"errors"
)

type Provider struct {
	apiKey  string
	baseURL string
}

func NewProvider(apiKey, baseURL string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

func (p *Provider) TestConnection(ctx context.Context) (bool, string, error) {
	return false, "Grok/xAI provider is not implemented for testing connection in Phase 2B.", nil
}

func (p *Provider) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	return "", errors.New("generative completion not active in Phase 2B")
}
