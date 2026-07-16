package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/admbahm/theForge/pkg/models"
)

func TestGeminiClient_GenerateIntel(t *testing.T) {
	client, err := newGeminiClient("test-key", "gemini-2.5-flash")
	if err != nil {
		t.Fatal(err)
	}

	client.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			expectedURL := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=test-key"
			if req.URL.String() != expectedURL {
				t.Fatalf("unexpected URL:\nexpected: %s\ngot:      %s", expectedURL, req.URL.String())
			}

			var body geminiRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body.Contents) != 1 || len(body.Contents[0].Parts) != 1 {
				t.Fatalf("unexpected contents structure: %+v", body.Contents)
			}

			resp := `{
				"candidates": [
					{
						"content": {
							"parts": [
								{
									"text": "### Role Summary\nMocked Gemini Intelligence."
								}
							]
						}
					}
				]
			}`
			return jsonResponse(http.StatusOK, resp), nil
		}),
	}

	res, err := client.GenerateIntel(context.Background(), models.JobPost{
		Company: "Oscorp",
		Title:   "Bio Engineer",
		Content: "Build things.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res != "### Role Summary\nMocked Gemini Intelligence." {
		t.Fatalf("unexpected response: %q", res)
	}
}
