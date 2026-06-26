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
