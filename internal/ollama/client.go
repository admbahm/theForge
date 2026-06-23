package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/admbahm/theForge/pkg/models"
)

const DefaultModel = "gemma4:e4b"

// Client generates job intelligence through the Ollama HTTP API.
type Client struct {
	baseURL    *url.URL
	model      string
	httpClient *http.Client
}

// NewClient creates an Ollama client.
func NewClient(baseURL, model string) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse Ollama API URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("Ollama API URL must use http or https")
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("Ollama API URL must include a host")
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultModel
	}

	return &Client{
		baseURL: parsedURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

type generateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

type generateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// GenerateIntel asks Ollama for concise Markdown intelligence about a job.
func (c *Client) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	requestBody := generateRequest{
		Model:  c.model,
		Prompt: buildPrompt(job),
		Stream: false,
		Options: map[string]any{
			"temperature": 0.2,
		},
	}

	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("encode Ollama request: %w", err)
	}

	endpoint := c.baseURL.JoinPath("api", "generate")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(encodedBody))
	if err != nil {
		return "", fmt.Errorf("create Ollama request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("call Ollama: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read Ollama response: %w", err)
	}

	var result generateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode Ollama response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if result.Error != "" {
			return "", fmt.Errorf("Ollama returned %s: %s", response.Status, result.Error)
		}
		return "", fmt.Errorf("Ollama returned %s", response.Status)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Ollama generation failed: %s", result.Error)
	}

	intel := strings.TrimSpace(result.Response)
	if intel == "" {
		return "", fmt.Errorf("Ollama returned an empty response")
	}
	intel = strings.TrimPrefix(intel, "```markdown")
	intel = strings.TrimPrefix(intel, "```")
	intel = strings.TrimSuffix(intel, "```")
	return strings.TrimSpace(intel), nil
}

func buildPrompt(job models.JobPost) string {
	return fmt.Sprintf(`You are producing evidence-based job intelligence for a candidate.

Return concise Markdown only, without a surrounding code fence. Use these headings:
### Role Summary
### Required Capabilities
### Candidate Positioning
### Risks and Unknowns
### Interview Themes

Do not invent facts. Clearly label missing information. Base the analysis only on the posting below.

Company: %s
Title: %s
Location: %s
Posted: %s

Posting:
%s`, job.Company, job.Title, job.Location, job.PostedAt, job.Content)
}
