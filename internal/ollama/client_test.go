package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/admbahm/theForge/pkg/models"
)

func TestGenerateIntelUsesConfiguredModel(t *testing.T) {
	client, err := NewClient("http://ollama.test", DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/api/generate" {
			t.Fatalf("path = %q, want /api/generate", request.URL.Path)
		}

		var body generateRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != DefaultModel {
			t.Fatalf("model = %q, want %q", body.Model, DefaultModel)
		}
		if body.Stream {
			t.Fatal("stream = true, want false")
		}
		if !strings.Contains(body.Prompt, "Senior Engineer") {
			t.Fatalf("prompt does not contain job title: %q", body.Prompt)
		}

		return jsonResponse(http.StatusOK, `{"response":"### Role Summary\nRelevant role."}`), nil
	})}

	intel, err := client.GenerateIntel(context.Background(), models.JobPost{
		Company: "Example",
		Title:   "Senior Engineer",
		Content: "Build reliable systems.",
	})
	if err != nil {
		t.Fatalf("GenerateIntel() error = %v", err)
	}
	if intel != "### Role Summary\nRelevant role." {
		t.Fatalf("intel = %q", intel)
	}
}

func TestBuildPromptRequiresTransferableFramingForUnsupportedAWS(t *testing.T) {
	prompt := buildPrompt(models.JobPost{
		Company: "Example",
		Title:   "Platform Engineer",
		Content: "Must have AWS production experience.",
	})

	for _, expected := range []string{
		"do not claim AWS production experience",
		"Frame cloud infrastructure skills as transferable",
		"mark AWS as a gap until verified",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func TestBuildPromptForbidsInventedMetrics(t *testing.T) {
	prompt := buildPrompt(models.JobPost{
		Company: "Example",
		Title:   "Reliability Engineer",
		Content: "Improve incident response and uptime.",
	})

	for _, expected := range []string{
		"Metrics may be used only when explicitly present",
		"do not invent percentages, dollar amounts, team sizes, uptime, or incident-reduction numbers",
		"Never fabricate employers, roles, dates, metrics",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func TestGenerateIntelReturnsAPIError(t *testing.T) {
	client, err := NewClient("http://ollama.test", DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{"error":"model not found"}`), nil
	})}

	_, err = client.GenerateIntel(context.Background(), models.JobPost{})
	if err == nil || !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("GenerateIntel() error = %v, want model error", err)
	}
}

func TestNewClientRejectsInvalidURL(t *testing.T) {
	if _, err := NewClient("file:///tmp/ollama", DefaultModel); err == nil {
		t.Fatal("NewClient() error = nil, want invalid URL error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
