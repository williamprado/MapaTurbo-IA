package domain

import "context"

type AINode struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parentId"` // pointer to support null
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Level    int     `json:"level"`
	Order    int     `json:"order"`
}

type AIEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type GenerateMindMapInput struct {
	Type     string
	Title    string
	Content  string
	Depth    int
	Language string
	Style    string
}

type MindMapAIResult struct {
	Title        string   `json:"title"`
	CentralTopic string   `json:"centralTopic"`
	Summary      string   `json:"summary"`
	Nodes        []AINode `json:"nodes"`
	Edges        []AIEdge `json:"edges"`
	RawPayload   string   `json:"-"`
}

type AIProvider interface {
	TestConnection(ctx context.Context) (bool, string, error)
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
	GenerateMindMap(ctx context.Context, input GenerateMindMapInput) (*MindMapAIResult, error)
}
