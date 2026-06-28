package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/admbahm/theForge/pkg/models"
)

const (
	DefaultModel = "gemma4:e4b"

	// Circuit Breaker settings
	maxFailures      = 3
	cooldownDuration = 30 * time.Second
)

type breakerState int

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

// Client generates job intelligence through the Ollama HTTP API.
type Client struct {
	baseURL    *url.URL
	model      string
	httpClient *http.Client

	// Circuit Breaker state
	mu              sync.Mutex
	state           breakerState
	failures        int
	lastStateChange time.Time

	// Cooldown override for tests
	cooldownOverride time.Duration
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
		state:           stateClosed,
		failures:        0,
		lastStateChange: time.Now(),
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

// GenerateIntel asks Ollama for concise, evidence-aware Markdown intelligence about a job.
func (c *Client) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	if err := c.checkBreaker(); err != nil {
		return "", err
	}

	requestBody := generateRequest{
		Model:  c.model,
		Prompt: buildPrompt(ctx, job),
		Stream: false,
		Options: map[string]any{
			"temperature": 0.2,
		},
	}

	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("encode Ollama request: %w", err)
	}

	// Enforce a strict 60-second limit for the individual HTTP request context
	callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	endpoint := c.baseURL.JoinPath("api", "generate")
	request, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint.String(), bytes.NewReader(encodedBody))
	if err != nil {
		return "", fmt.Errorf("create Ollama request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		c.recordFailure()
		return "", fmt.Errorf("call Ollama: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		c.recordFailure()
		return "", fmt.Errorf("read Ollama response: %w", err)
	}

	var result generateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		c.recordFailure()
		return "", fmt.Errorf("decode Ollama response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		c.recordFailure()
		if result.Error != "" {
			return "", fmt.Errorf("Ollama returned %s: %s", response.Status, result.Error)
		}
		return "", fmt.Errorf("Ollama returned %s", response.Status)
	}
	if result.Error != "" {
		c.recordFailure()
		return "", fmt.Errorf("Ollama generation failed: %s", result.Error)
	}

	c.recordSuccess()

	intel := strings.TrimSpace(result.Response)
	if intel == "" {
		return "", fmt.Errorf("Ollama returned an empty response")
	}
	intel = strings.TrimPrefix(intel, "```markdown")
	intel = strings.TrimPrefix(intel, "```")
	intel = strings.TrimSuffix(intel, "```")
	return strings.TrimSpace(intel), nil
}

func (c *Client) checkBreaker() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cooldown := cooldownDuration
	if c.cooldownOverride > 0 {
		cooldown = c.cooldownOverride
	}

	if c.state == stateOpen {
		if time.Since(c.lastStateChange) > cooldown {
			c.state = stateHalfOpen
			c.lastStateChange = time.Now()
			return nil
		}
		return errors.New("circuit breaker open: ollama server is currently unavailable")
	}
	return nil
}

func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures = 0
	if c.state != stateClosed {
		c.state = stateClosed
		c.lastStateChange = time.Now()
	}
}

func (c *Client) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures++
	if c.state == stateHalfOpen || c.failures >= maxFailures {
		c.state = stateOpen
		c.lastStateChange = time.Now()
	}
}

func buildPrompt(ctx context.Context, job models.JobPost) string {
	tier, _ := ctx.Value("tier").(string)
	if tier == "local" {
		return buildLocalPrompt(job)
	}
	return buildFrontierPrompt(job)
}

func buildLocalPrompt(job models.JobPost) string {
	return fmt.Sprintf(`You are producing local baseline career intelligence for an ethical, evidence-based AI-assisted job application workflow.

Extract the core signals from the job posting below. Keep it concise, structured, and focused. Do not invent details.

Return concise Markdown only, without a surrounding code fence. Use these headings:
### Company Profile
### Role Summary
### Key Requirements & Tech Stack
### Keyword Signals

Company: %s
Title: %s
Location: %s
Posted: %s

Posting:
%s`, job.Company, job.Title, job.Location, job.PostedAt, job.Content)
}

func buildFrontierPrompt(job models.JobPost) string {
	return fmt.Sprintf(`You are producing career intelligence for an ethical, evidence-based AI-assisted job application workflow.

The Forge is not a generic resume generator. Its core rule is strict evidence discipline:
- Never invent candidate experience.
- Never fabricate employers, roles, dates, metrics, technologies, certifications, education, clearance status, citizenship, accomplishments, or production experience.
- Parse the job posting into requirements, responsibilities, keywords, domain signals, and implied expectations.
- Treat unsupported requirements as gaps or transferable skills, not as direct experience.
- Metrics may be used only when explicitly present in verified source material.
- Approximate or inferred claims must be labeled as inferred internally and must not appear as hard facts.
- Prefer concrete evidence and hiring-manager usefulness over keyword stuffing.

Verified candidate source material is not included in this phase. Base this intelligence only on the job posting below and identify what evidence should be requested from the candidate before application materials are generated.

Important examples:
- If a posting asks for AWS but verified candidate evidence only shows GCP, Kubernetes, or Terraform, do not claim AWS production experience. Frame cloud infrastructure skills as transferable and mark AWS as a gap until verified.
- If a metric is missing, do not invent percentages, dollar amounts, team sizes, uptime, or incident-reduction numbers.

Return concise Markdown only, without a surrounding code fence. Use these headings:
### Role Summary
### Requirement Signals
### Evidence Needed
### Transferable Positioning
### Gaps and Unsupported Claims
### Candidate Follow-Up Questions
### Interview Themes

Keep the tone clear, ethical, candidate-first, hiring-manager aware, and anti-keyword-stuffing.

Company: %s
Title: %s
Location: %s
Posted: %s

Posting:
%s`, job.Company, job.Title, job.Location, job.PostedAt, job.Content)
}
