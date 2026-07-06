package ai

import "context"

type AIProvider interface {
	TestConnection(ctx context.Context) (bool, string, error)
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
}
