package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/admbahm/theForge/internal/ollama"
	"github.com/admbahm/theForge/pkg/models"
)

type openaiClient struct {
	apiKey string
	model  string
	client *http.Client
}

func newOpenAIClient(apiKey, model string) (*openaiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai client: API key is required")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &openaiClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}, nil
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float32         `json:"temperature,omitempty"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *openaiClient) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	prompt := ollama.BuildPrompt(ctx, job)
	reqBody := openaiRequest{
		Model: c.model,
		Messages: []openaiMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("openai: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errRes openaiResponse
		if err := json.Unmarshal(body, &errRes); err == nil && errRes.Error != nil {
			return "", fmt.Errorf("openai error: %s (status %d)", errRes.Error.Message, resp.StatusCode)
		}
		return "", fmt.Errorf("openai API returned status %d: %s", resp.StatusCode, string(body))
	}

	var res openaiResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("openai: unmarshal response: %w", err)
	}

	if len(res.Choices) == 0 {
		return "", fmt.Errorf("openai: empty choices returned")
	}

	intel := strings.TrimSpace(res.Choices[0].Message.Content)
	intel = strings.TrimPrefix(intel, "```markdown")
	intel = strings.TrimPrefix(intel, "```")
	intel = strings.TrimSuffix(intel, "```")
	return strings.TrimSpace(intel), nil
}
