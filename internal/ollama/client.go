package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

// VerifyModelAvailability checks if the model exists in the local pulled tags.
func (c *Client) VerifyModelAvailability(ctx context.Context, model string) (bool, error) {
	if model == "" {
		model = c.model
	}
	models, err := c.ListLocalModels(ctx)
	if err != nil {
		return false, fmt.Errorf("list local models: %w", err)
	}
	for _, m := range models {
		if strings.EqualFold(m.Name, model) || strings.EqualFold(m.Model, model) {
			return true, nil
		}
	}
	return false, nil
}

// OptimizeVRAM unloads any conflicting active model currently in VRAM.
func (c *Client) OptimizeVRAM(ctx context.Context, targetModel string) error {
	if targetModel == "" {
		targetModel = c.model
	}
	active, err := c.ListActiveModels(ctx)
	if err != nil {
		return fmt.Errorf("list active models: %w", err)
	}

	for _, m := range active {
		if !strings.EqualFold(m.Name, targetModel) && !strings.EqualFold(m.Model, targetModel) {
			log.Printf("[VRAM] Conflicting model %q is active in memory. Unloading to free VRAM...", m.Name)
			if err := c.UnloadModel(ctx, m.Name); err != nil {
				log.Printf("[Warning] Failed to unload model %q: %v", m.Name, err)
			}
		}
	}
	return nil
}

// ModelDetails represents the details of an Ollama model.
type ModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ModelInfo represents metadata of an Ollama model.
type ModelInfo struct {
	Name      string       `json:"name"`
	Model     string       `json:"model"`
	Size      int64        `json:"size"`
	Digest    string       `json:"digest"`
	Details   ModelDetails `json:"details"`
	SizeVRAM  int64        `json:"size_vram,omitempty"`
	ExpiresAt string       `json:"expires_at,omitempty"`
}

type tagsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ListLocalModels retrieves all pulled/stored models from local disk.
func (c *Client) ListLocalModels(ctx context.Context) ([]ModelInfo, error) {
	endpoint := c.baseURL.JoinPath("api", "tags")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create tags request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute tags request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama tags returned status %d", resp.StatusCode)
	}

	var res tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode tags response: %w", err)
	}
	return res.Models, nil
}

// ListActiveModels retrieves all models currently active/loaded in RAM/VRAM.
func (c *Client) ListActiveModels(ctx context.Context) ([]ModelInfo, error) {
	endpoint := c.baseURL.JoinPath("api", "ps")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create ps request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute ps request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama ps returned status %d", resp.StatusCode)
	}

	var res tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode ps response: %w", err)
	}
	return res.Models, nil
}

// UnloadModel unloads a model from memory (RAM/VRAM) immediately.
func (c *Client) UnloadModel(ctx context.Context, modelName string) error {
	endpoint := c.baseURL.JoinPath("api", "generate")
	body := map[string]any{
		"model":      modelName,
		"keep_alive": 0,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal unload request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create unload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute unload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unload returned status %d", resp.StatusCode)
	}
	return nil
}
