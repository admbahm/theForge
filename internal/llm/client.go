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

type clientWrapper struct {
	Client
	maxContextLength int
}

func (w *clientWrapper) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	job.Content = TruncateContext(job.Content, w.maxContextLength)
	return w.Client.GenerateIntel(ctx, job)
}

// NewClient constructs the configured LLM provider client.
func NewClient(cfg config.Config) (Client, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	if provider == "" {
		provider = config.DefaultLLMProvider
	}

	var client Client
	var err error

	switch provider {
	case "ollama":
		host := firstConfigured(cfg.Providers.Ollama.Host, cfg.OllamaAPIURL, config.DefaultOllamaAPIURL)
		model := firstConfigured(cfg.LLM.Model, cfg.Providers.Ollama.Model, cfg.OllamaModel, config.DefaultOllamaModel)
		client, err = ollama.NewClient(host, model)
		if err != nil {
			return nil, fmt.Errorf("create Ollama client: %w", err)
		}
	case "openai":
		client, err = newStubClient("openai", firstConfigured(cfg.LLM.Model, cfg.Providers.OpenAI.Model, config.DefaultOpenAIModel), firstConfigured(cfg.Providers.OpenAI.APIKeyEnv, config.DefaultOpenAIKeyEnv))
		if err != nil {
			return nil, err
		}
	case "gemini":
		client, err = newStubClient("gemini", firstConfigured(cfg.LLM.Model, cfg.Providers.Gemini.Model, config.DefaultGeminiModel), firstConfigured(cfg.Providers.Gemini.APIKeyEnv, config.DefaultGeminiKeyEnv))
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported LLM provider %q (supported: ollama, openai, gemini)", provider)
	}

	maxLen := cfg.MaxContextLength
	if maxLen <= 0 {
		maxLen = config.DefaultMaxContextLength
	}

	return &clientWrapper{
		Client:           client,
		maxContextLength: maxLen,
	}, nil
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
