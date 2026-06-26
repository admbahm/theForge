package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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

func TestCircuitBreakerTripsAndRecovers(t *testing.T) {
	client, err := NewClient("http://ollama.test", DefaultModel)
	if err != nil {
		t.Fatal(err)
	}

	// Make the cooldown very short for testing
	client.cooldownOverride = 10 * time.Millisecond

	// 1. Simulate 3 sequential failures to trip the circuit breaker
	client.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusInternalServerError, `{"error":"internal error"}`), nil
	})}

	for i := 0; i < 3; i++ {
		_, err = client.GenerateIntel(context.Background(), models.JobPost{})
		if err == nil || !strings.Contains(err.Error(), "internal error") {
			t.Fatalf("[%d] expected internal error, got: %v", i, err)
		}
	}

	// 2. Next request should fail immediately via the circuit breaker (no HTTP requests made)
	client.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		t.Fatal("no requests should be made when circuit breaker is open")
		return nil, nil
	})}

	_, err = client.GenerateIntel(context.Background(), models.JobPost{})
	if err == nil || !strings.Contains(err.Error(), "circuit breaker open") {
		t.Fatalf("expected circuit breaker open error, got: %v", err)
	}

	// 3. Sleep past cooldown to transition into half-open state
	time.Sleep(15 * time.Millisecond)

	// 4. Have the probe request succeed. This should recover the circuit breaker to closed.
	client.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"response":"### Role Summary\nRecovered."}`), nil
	})}

	intel, err := client.GenerateIntel(context.Background(), models.JobPost{})
	if err != nil {
		t.Fatalf("expected successful recovery call, got error: %v", err)
	}
	if intel != "### Role Summary\nRecovered." {
		t.Fatalf("unexpected response: %q", intel)
	}

	// 5. Subsequent request should succeed normally
	intel, err = client.GenerateIntel(context.Background(), models.JobPost{})
	if err != nil {
		t.Fatalf("expected subsequent call to succeed, got error: %v", err)
	}
	if intel != "### Role Summary\nRecovered." {
		t.Fatalf("unexpected response: %q", intel)
	}
}

func TestGenerateIntelTimeout(t *testing.T) {
	client, err := NewClient("http://ollama.test", DefaultModel)
	if err != nil {
		t.Fatal(err)
	}

	// Trigger timeout via context cancellation or HTTP hang
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		// Verify context has a timeout deadline set
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("expected request context to have a deadline")
		}
		return nil, context.DeadlineExceeded
	})}

	_, err = client.GenerateIntel(context.Background(), models.JobPost{})
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected context deadline exceeded error, got: %v", err)
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
