// Package llm selects provider-specific clients behind a provider-neutral interface.
package llm

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/admbahm/theForge/internal/config"
	"github.com/admbahm/theForge/internal/ollama"
	"github.com/admbahm/theForge/pkg/models"
)

// Client generates Markdown intelligence for a job posting.
// Its method set intentionally matches engine.IntelGenerator.
type Client interface {
	GenerateIntel(ctx context.Context, job models.JobPost) (string, error)
}

// ModelManager verifies model presence and handles active swaps for memory budgeting.
type ModelManager interface {
	VerifyModelAvailability(ctx context.Context, model string) (bool, error)
	OptimizeVRAM(ctx context.Context, targetModel string) error
}

type clientWrapper struct {
	Client
	localClient      Client
	maxContextLength int
}

func (w *clientWrapper) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	job.Content = TruncateContext(ctx, w.localClient, job.Content, w.maxContextLength)
	return w.Client.GenerateIntel(ctx, job)
}

func (w *clientWrapper) VerifyModelAvailability(ctx context.Context, model string) (bool, error) {
	if mm, ok := w.Client.(ModelManager); ok {
		return mm.VerifyModelAvailability(ctx, model)
	}
	return true, nil
}

func (w *clientWrapper) OptimizeVRAM(ctx context.Context, targetModel string) error {
	if mm, ok := w.Client.(ModelManager); ok {
		return mm.OptimizeVRAM(ctx, targetModel)
	}
	return nil
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

func (r *routingClient) VerifyModelAvailability(ctx context.Context, model string) (bool, error) {
	if mm, ok := r.localClient.(ModelManager); ok {
		if ok, err := mm.VerifyModelAvailability(ctx, model); err != nil || !ok {
			return ok, err
		}
	}
	if mm, ok := r.frontierClient.(ModelManager); ok {
		if ok, err := mm.VerifyModelAvailability(ctx, model); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

func (r *routingClient) OptimizeVRAM(ctx context.Context, targetModel string) error {
	if mm, ok := r.localClient.(ModelManager); ok {
		if err := mm.OptimizeVRAM(ctx, targetModel); err != nil {
			return err
		}
	}
	if mm, ok := r.frontierClient.(ModelManager); ok {
		if err := mm.OptimizeVRAM(ctx, targetModel); err != nil {
			return err
		}
	}
	return nil
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

	// Verify local model is pulled/available (2s timeout to avoid blocking startup)
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer verifyCancel()
	if available, err := localClient.VerifyModelAvailability(verifyCtx, localModel); err == nil && !available {
		log.Printf("[Warning] Local model %q is not pulled in Ollama. Run 'ollama pull %s' to download it.", localModel, localModel)
	} else if err != nil {
		log.Printf("[Warning] Failed to verify local Ollama model availability: %v (is Ollama running?)", err)
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
		localClient:      localClient,
		maxContextLength: maxLen,
	}

	wrappedFrontier := &clientWrapper{
		Client:           frontierClient,
		localClient:      localClient,
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
