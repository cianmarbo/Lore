package llm

import "context"

// Provider is the interface for LLM text generation.
type Provider interface {
	Generate(ctx context.Context, req Request) (string, error)
}

// Request describes a single LLM generation request.
type Request struct {
	SystemPrompt string
	UserContent  string
	MaxTokens    int
}

// Config holds connection details for an LLM provider.
type Config struct {
	Provider string // "anthropic" or "openai"
	APIKey   string
	Model    string
	BaseURL  string // optional override
}

// NewProvider creates a Provider from config, returning nil if not configured.
func NewProvider(cfg Config) Provider {
	if cfg.APIKey == "" {
		return nil
	}
	switch cfg.Provider {
	case "anthropic":
		return newAnthropic(cfg)
	case "openai":
		return newOpenAI(cfg)
	default:
		return nil
	}
}

// Available returns true if the provider is non-nil.
func Available(p Provider) bool {
	return p != nil
}
