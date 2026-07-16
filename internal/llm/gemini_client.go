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

type geminiClient struct {
	apiKey string
	model  string
	client *http.Client
}

func newGeminiClient(apiKey, model string) (*geminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini client: API key is required")
	}
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &geminiClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}, nil
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature float32 `json:"temperature,omitempty"`
}

type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func (c *geminiClient) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	prompt := ollama.BuildPrompt(ctx, job)
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			Temperature: 0.2,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("gemini: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errRes geminiResponse
		if err := json.Unmarshal(body, &errRes); err == nil && errRes.Error != nil {
			return "", fmt.Errorf("gemini error: %s (status %d)", errRes.Error.Message, resp.StatusCode)
		}
		return "", fmt.Errorf("gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	var res geminiResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("gemini: unmarshal response: %w", err)
	}

	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty content/parts returned")
	}

	intel := strings.TrimSpace(res.Candidates[0].Content.Parts[0].Text)
	intel = strings.TrimPrefix(intel, "```markdown")
	intel = strings.TrimPrefix(intel, "```")
	intel = strings.TrimSuffix(intel, "```")
	return strings.TrimSpace(intel), nil
}
