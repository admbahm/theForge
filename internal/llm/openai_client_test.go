package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/admbahm/theForge/pkg/models"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestOpenAIClient_GenerateIntel(t *testing.T) {
	client, err := newOpenAIClient("test-key", "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}

	client.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://api.openai.com/v1/chat/completions" {
				t.Fatalf("unexpected URL: %s", req.URL.String())
			}
			if req.Header.Get("Authorization") != "Bearer test-key" {
				t.Fatalf("unexpected Authorization header: %s", req.Header.Get("Authorization"))
			}

			var body openaiRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Model != "gpt-4o" {
				t.Fatalf("unexpected model: %s", body.Model)
			}
			if len(body.Messages) != 1 || body.Messages[0].Role != "user" {
				t.Fatalf("unexpected messages: %+v", body.Messages)
			}

			resp := `{
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": "### Role Summary\nMocked OpenAI Intelligence."
						}
					}
				]
			}`
			return jsonResponse(http.StatusOK, resp), nil
		}),
	}

	res, err := client.GenerateIntel(context.Background(), models.JobPost{
		Company: "Stark Industries",
		Title:   "Software Engineer",
		Content: "Build things.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res != "### Role Summary\nMocked OpenAI Intelligence." {
		t.Fatalf("unexpected response: %q", res)
	}
}
