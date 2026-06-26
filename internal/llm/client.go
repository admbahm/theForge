// Package llm selects provider-specific clients behind a provider-neutral interface.
package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/admbahm/theForge/internal/config"
	"github.com/admbahm/theForge/internal/ollama"
	"github.com/admbahm/theForge/pkg/models"
)

// Client generates Markdown intelligence for a job posting.
// Its method set intentionally matches engine.IntelGenerator.
type Client interface {
	GenerateIntel(ctx context.Context, job models.JobPost) (string, error)
}

// NewClient constructs the configured LLM provider client.
func NewClient(cfg config.Config) (Client, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	if provider == "" {
		provider = config.DefaultLLMProvider
	}

	switch provider {
	case "ollama":
		host := firstConfigured(cfg.Providers.Ollama.Host, cfg.OllamaAPIURL, config.DefaultOllamaAPIURL)
		model := firstConfigured(cfg.LLM.Model, cfg.Providers.Ollama.Model, cfg.OllamaModel, config.DefaultOllamaModel)
		client, err := ollama.NewClient(host, model)
		if err != nil {
			return nil, fmt.Errorf("create Ollama client: %w", err)
		}
		return client, nil
	case "openai":
		return newStubClient("openai", firstConfigured(cfg.LLM.Model, cfg.Providers.OpenAI.Model, config.DefaultOpenAIModel), firstConfigured(cfg.Providers.OpenAI.APIKeyEnv, config.DefaultOpenAIKeyEnv))
	case "gemini":
		return newStubClient("gemini", firstConfigured(cfg.LLM.Model, cfg.Providers.Gemini.Model, config.DefaultGeminiModel), firstConfigured(cfg.Providers.Gemini.APIKeyEnv, config.DefaultGeminiKeyEnv))
	default:
		return nil, fmt.Errorf("unsupported LLM provider %q (supported: ollama, openai, gemini)", provider)
	}
}

func firstConfigured(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

type stubClient struct {
	provider string
	model    string
}

func newStubClient(provider, model, apiKeyEnv string) (Client, error) {
	if strings.TrimSpace(os.Getenv(apiKeyEnv)) == "" {
		return nil, fmt.Errorf("%s provider requires an API key in environment variable %s", provider, apiKeyEnv)
	}
	return &stubClient{provider: provider, model: model}, nil
}

func (c *stubClient) GenerateIntel(context.Context, models.JobPost) (string, error) {
	// TODO: Implement the provider HTTP API without changing the Client contract.
	return "", fmt.Errorf("%s provider is configured with model %q but generation is not implemented yet", c.provider, c.model)
}
