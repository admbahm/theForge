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

type routingClient struct {
	localClient    Client
	frontierClient Client
}

func (r *routingClient) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	tier := "frontier" // Default
	if val, ok := ctx.Value("tier").(string); ok {
		tier = val
	}

	if tier == "local" {
		if r.localClient == nil {
			return "", fmt.Errorf("local tier LLM client is not initialized")
		}
		return r.localClient.GenerateIntel(ctx, job)
	}

	if r.frontierClient == nil {
		return "", fmt.Errorf("frontier tier LLM client is not initialized")
	}
	return r.frontierClient.GenerateIntel(ctx, job)
}

// NewClient constructs the configured LLM provider client.
func NewClient(cfg config.Config) (Client, error) {
	// 1. Always initialize local client (Ollama)
	localHost := firstConfigured(cfg.Providers.Ollama.Host, cfg.OllamaAPIURL, config.DefaultOllamaAPIURL)
	localModel := firstConfigured(cfg.Providers.Ollama.Model, cfg.OllamaModel, config.DefaultOllamaModel)
	localClient, err := ollama.NewClient(localHost, localModel)
	if err != nil {
		return nil, fmt.Errorf("create local Ollama client: %w", err)
	}

	// 2. Initialize frontier client based on configured provider
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	if provider == "" {
		provider = config.DefaultLLMProvider
	}

	var frontierClient Client
	switch provider {
	case "ollama":
		frontierClient = localClient
	case "openai":
		frontierClient, err = newStubClient("openai", firstConfigured(cfg.LLM.Model, cfg.Providers.OpenAI.Model, config.DefaultOpenAIModel), firstConfigured(cfg.Providers.OpenAI.APIKeyEnv, config.DefaultOpenAIKeyEnv))
		if err != nil {
			return nil, err
		}
	case "gemini":
		frontierClient, err = newStubClient("gemini", firstConfigured(cfg.LLM.Model, cfg.Providers.Gemini.Model, config.DefaultGeminiModel), firstConfigured(cfg.Providers.Gemini.APIKeyEnv, config.DefaultGeminiKeyEnv))
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

	wrappedLocal := &clientWrapper{
		Client:           localClient,
		maxContextLength: maxLen,
	}

	wrappedFrontier := &clientWrapper{
		Client:           frontierClient,
		maxContextLength: maxLen,
	}

	return &routingClient{
		localClient:    wrappedLocal,
		frontierClient: wrappedFrontier,
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
