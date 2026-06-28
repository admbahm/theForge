package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/admbahm/theForge/internal/config"
	"github.com/admbahm/theForge/pkg/models"
)

func TestNewClientDefaultsToOllamaWithoutAPIKey(t *testing.T) {
	t.Setenv(config.DefaultOpenAIKeyEnv, "")
	t.Setenv(config.DefaultGeminiKeyEnv, "")

	client, err := NewClient(config.Config{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() client = nil")
	}
}

func TestNewClientRequiresKeyOnlyForSelectedProvider(t *testing.T) {
	t.Setenv(config.DefaultOpenAIKeyEnv, "")

	_, err := NewClient(config.Config{LLM: config.LLMConfig{Provider: "openai"}})
	if err == nil || !strings.Contains(err.Error(), config.DefaultOpenAIKeyEnv) {
		t.Fatalf("NewClient() error = %v, want missing key error", err)
	}
}

func TestNewClientCreatesConfiguredProviderStub(t *testing.T) {
	t.Setenv("CUSTOM_GEMINI_KEY", "test-key")
	cfg := config.Config{
		LLM: config.LLMConfig{Provider: "gemini", Model: "custom-model"},
		Providers: config.ProvidersConfig{
			Gemini: config.APIConfig{APIKeyEnv: "CUSTOM_GEMINI_KEY"},
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.GenerateIntel(context.Background(), models.JobPost{})
	if err == nil || !strings.Contains(err.Error(), "gemini provider") || !strings.Contains(err.Error(), "custom-model") {
		t.Fatalf("GenerateIntel() error = %v, want useful stub error", err)
	}
}

func TestNewClientRejectsUnknownProvider(t *testing.T) {
	_, err := NewClient(config.Config{LLM: config.LLMConfig{Provider: "unknown"}})
	if err == nil || !strings.Contains(err.Error(), "unsupported LLM provider") {
		t.Fatalf("NewClient() error = %v, want unsupported provider error", err)
	}
}

type fakeInnerClient struct {
	lastJob models.JobPost
}

func (f *fakeInnerClient) GenerateIntel(_ context.Context, job models.JobPost) (string, error) {
	f.lastJob = job
	return "done", nil
}

func TestClientWrapperTruncatesContext(t *testing.T) {
	inner := &fakeInnerClient{}
	wrapper := &clientWrapper{
		Client:           inner,
		maxContextLength: 30,
	}

	content := "This is a very long job description that will be truncated."
	_, err := wrapper.GenerateIntel(context.Background(), models.JobPost{
		Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(inner.lastJob.Content) > 30 {
		t.Fatalf("expected job description to be truncated, got length %d: %q", len(inner.lastJob.Content), inner.lastJob.Content)
	}
}

type fakeRoutingClient struct {
	calledWith string
}

func (f *fakeRoutingClient) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	return f.calledWith, nil
}

func TestRoutingClientRoutesCorrectly(t *testing.T) {
	local := &fakeRoutingClient{calledWith: "local"}
	frontier := &fakeRoutingClient{calledWith: "frontier"}

	rc := &routingClient{
		localClient:    local,
		frontierClient: frontier,
	}

	// 1. Without context value, defaults to frontier
	res, err := rc.GenerateIntel(context.Background(), models.JobPost{})
	if err != nil {
		t.Fatal(err)
	}
	if res != "frontier" {
		t.Fatalf("expected 'frontier', got %q", res)
	}

	// 2. With local context value
	localCtx := context.WithValue(context.Background(), "tier", "local")
	res, err = rc.GenerateIntel(localCtx, models.JobPost{})
	if err != nil {
		t.Fatal(err)
	}
	if res != "local" {
		t.Fatalf("expected 'local', got %q", res)
	}

	// 3. With frontier context value
	frontierCtx := context.WithValue(context.Background(), "tier", "frontier")
	res, err = rc.GenerateIntel(frontierCtx, models.JobPost{})
	if err != nil {
		t.Fatal(err)
	}
	if res != "frontier" {
		t.Fatalf("expected 'frontier', got %q", res)
	}
}
