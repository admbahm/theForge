package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	openHuntOutputDirKey = "OPENHUNT_OUTPUT_DIR"
	llmProviderKey       = "LLM_PROVIDER"
	llmModelKey          = "LLM_MODEL"
	ollamaAPIURLKey      = "OLLAMA_API_URL"
	ollamaModelKey       = "OLLAMA_MODEL"
	openAIAPIKeyEnvKey   = "OPENAI_API_KEY_ENV"
	openAIModelKey       = "OPENAI_MODEL"
	geminiAPIKeyEnvKey   = "GEMINI_API_KEY_ENV"
	geminiModelKey       = "GEMINI_MODEL"
	ckbDirKey            = "THEFORGE_CKB_DIR"
	demoModeKey          = "THEFORGE_DEMO_MODE"
	contactNameKey       = "THEFORGE_CONTACT_NAME"
	contactEmailKey      = "THEFORGE_CONTACT_EMAIL"
	contactPhoneKey      = "THEFORGE_CONTACT_PHONE"
	contactAddressKey    = "THEFORGE_CONTACT_ADDRESS"
	contactLinkedInKey   = "THEFORGE_CONTACT_LINKEDIN"
	contactGitHubKey     = "THEFORGE_CONTACT_GITHUB"

	DefaultLLMProvider      = "ollama"
	DefaultOllamaAPIURL     = "http://localhost:11434"
	DefaultOllamaModel      = "gemma4:e4b"
	DefaultOpenAIKeyEnv     = "OPENAI_API_KEY"
	DefaultOpenAIModel      = "gpt-4.1-mini"
	DefaultGeminiKeyEnv     = "GEMINI_API_KEY"
	DefaultGeminiModel      = "gemini-2.5-flash"
	DefaultConcurrency      = 4
	DefaultMaxContextLength = 24000

	concurrencyKey      = "THEFORGE_CONCURRENCY"
	maxContextLengthKey = "THEFORGE_MAX_CONTEXT_LENGTH"
)

// Config contains the runtime configuration for The Forge.
type Config struct {
	OpenHuntOutputDir string            `yaml:"openhunt_output_dir"`
	Concurrency       int               `yaml:"concurrency"`
	MaxContextLength  int               `yaml:"max_context_length"`
	LLM               LLMConfig         `yaml:"llm"`
	Providers         ProvidersConfig   `yaml:"providers"`
	Application       ApplicationConfig `yaml:"application"`

	// OllamaAPIURL and OllamaModel preserve the original programmatic config API.
	OllamaAPIURL string `yaml:"-"`
	OllamaModel  string `yaml:"-"`
}

type ApplicationConfig struct {
	CKBDir   string        `yaml:"ckb_dir"`
	DemoMode bool          `yaml:"demo_mode"`
	Contact  ContactConfig `yaml:"contact"`
}

type ContactConfig struct {
	Name        string `yaml:"name"`
	Email       string `yaml:"email"`
	Phone       string `yaml:"phone"`
	Address     string `yaml:"address"`
	LinkedInURL string `yaml:"linkedin"`
	GitHubURL   string `yaml:"github"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

type ProvidersConfig struct {
	Ollama OllamaConfig `yaml:"ollama"`
	OpenAI APIConfig    `yaml:"openai"`
	Gemini APIConfig    `yaml:"gemini"`
}

type OllamaConfig struct {
	Host  string `yaml:"host"`
	Model string `yaml:"model"`
}

type APIConfig struct {
	APIKeyEnv string `yaml:"api_key_env"`
	Model     string `yaml:"model"`
}

// Load reads optional dotenv values and validates the runtime configuration.
// Values already present in the process environment take precedence.
func Load(dotenvPath string, yamlPaths ...string) (Config, error) {
	if err := loadDotEnv(dotenvPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load %s: %w", dotenvPath, err)
	}

	cfg := defaultConfig()
	for _, path := range yamlPaths {
		if err := loadYAML(path, &cfg); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("load %s: %w", path, err)
		}
	}
	applyEnvironment(&cfg)

	outputDir := strings.TrimSpace(cfg.OpenHuntOutputDir)
	if outputDir == "" {
		return Config{}, fmt.Errorf("%s is required", openHuntOutputDirKey)
	}

	absoluteDir, err := filepath.Abs(outputDir)
	if err != nil {
		return Config{}, fmt.Errorf("resolve %s: %w", openHuntOutputDirKey, err)
	}

	info, err := os.Stat(absoluteDir)
	if err != nil {
		return Config{}, fmt.Errorf("validate %s %q: %w", openHuntOutputDirKey, absoluteDir, err)
	}
	if !info.IsDir() {
		return Config{}, fmt.Errorf("%s %q is not a directory", openHuntOutputDirKey, absoluteDir)
	}

	cfg.OpenHuntOutputDir = absoluteDir
	if err := resolveApplicationConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.OllamaAPIURL = cfg.Providers.Ollama.Host
	cfg.OllamaModel = cfg.Providers.Ollama.Model
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Concurrency:      DefaultConcurrency,
		MaxContextLength: DefaultMaxContextLength,
		LLM:              LLMConfig{Provider: DefaultLLMProvider},
		Providers: ProvidersConfig{
			Ollama: OllamaConfig{Host: DefaultOllamaAPIURL, Model: DefaultOllamaModel},
			OpenAI: APIConfig{APIKeyEnv: DefaultOpenAIKeyEnv, Model: DefaultOpenAIModel},
			Gemini: APIConfig{APIKeyEnv: DefaultGeminiKeyEnv, Model: DefaultGeminiModel},
		},
	}
}

func loadYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}
	return nil
}

func applyEnvironment(cfg *Config) {
	setFromEnvironment(&cfg.OpenHuntOutputDir, openHuntOutputDirKey)
	if value, exists := os.LookupEnv(concurrencyKey); exists && strings.TrimSpace(value) != "" {
		if val, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && val > 0 {
			cfg.Concurrency = val
		}
	}
	if value, exists := os.LookupEnv(maxContextLengthKey); exists && strings.TrimSpace(value) != "" {
		if val, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && val > 0 {
			cfg.MaxContextLength = val
		}
	}
	setFromEnvironment(&cfg.LLM.Provider, llmProviderKey)
	setFromEnvironment(&cfg.LLM.Model, llmModelKey)
	setFromEnvironment(&cfg.Providers.Ollama.Host, ollamaAPIURLKey)
	setFromEnvironment(&cfg.Providers.Ollama.Model, ollamaModelKey)
	setFromEnvironment(&cfg.Providers.OpenAI.APIKeyEnv, openAIAPIKeyEnvKey)
	setFromEnvironment(&cfg.Providers.OpenAI.Model, openAIModelKey)
	setFromEnvironment(&cfg.Providers.Gemini.APIKeyEnv, geminiAPIKeyEnvKey)
	setFromEnvironment(&cfg.Providers.Gemini.Model, geminiModelKey)
	setFromEnvironment(&cfg.Application.CKBDir, ckbDirKey)
	if value, exists := os.LookupEnv(demoModeKey); exists && strings.TrimSpace(value) != "" {
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err == nil {
			cfg.Application.DemoMode = parsed
		}
	}
	setFromEnvironment(&cfg.Application.Contact.Name, contactNameKey)
	setFromEnvironment(&cfg.Application.Contact.Email, contactEmailKey)
	setFromEnvironment(&cfg.Application.Contact.Phone, contactPhoneKey)
	setFromEnvironment(&cfg.Application.Contact.Address, contactAddressKey)
	setFromEnvironment(&cfg.Application.Contact.LinkedInURL, contactLinkedInKey)
	setFromEnvironment(&cfg.Application.Contact.GitHubURL, contactGitHubKey)
}

func resolveApplicationConfig(cfg *Config) error {
	ckbDir := strings.TrimSpace(cfg.Application.CKBDir)
	if ckbDir == "" {
		return nil
	}
	absoluteDir, err := filepath.Abs(ckbDir)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", ckbDirKey, err)
	}
	info, err := os.Stat(absoluteDir)
	if err != nil {
		return fmt.Errorf("validate %s %q: %w", ckbDirKey, absoluteDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %q is not a directory", ckbDirKey, absoluteDir)
	}
	exampleDir := bundledExampleCKBDir()
	if exampleDir != "" && samePath(absoluteDir, exampleDir) && !cfg.Application.DemoMode {
		return fmt.Errorf("%s points to the bundled fictional example CKB; set %s=true only for explicit demos", ckbDirKey, demoModeKey)
	}
	cfg.Application.CKBDir = absoluteDir
	return nil
}

func bundledExampleCKBDir() string {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "ckb"))
}

func samePath(left, right string) bool {
	leftResolved, leftErr := filepath.EvalSymlinks(left)
	rightResolved, rightErr := filepath.EvalSymlinks(right)
	if leftErr == nil {
		left = leftResolved
	}
	if rightErr == nil {
		right = rightResolved
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func setFromEnvironment(target *string, key string) {
	if value, exists := os.LookupEnv(key); exists && strings.TrimSpace(value) != "" {
		*target = strings.TrimSpace(value)
	}
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		key, value, ok, err := parseDotEnvLine(scanner.Text())
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNumber, err)
		}
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read dotenv file: %w", err)
	}
	return nil
}

func parseDotEnvLine(line string) (string, string, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}

	line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
	key, value, found := strings.Cut(line, "=")
	if !found {
		return "", "", false, fmt.Errorf("expected KEY=VALUE")
	}

	key = strings.TrimSpace(key)
	if !validEnvironmentKey(key) {
		return "", "", false, fmt.Errorf("invalid environment variable name %q", key)
	}

	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", "", false, fmt.Errorf("invalid quoted value for %s: %w", key, err)
		}
		value = unquoted
	} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}

	return key, value, true, nil
}

func validEnvironmentKey(key string) bool {
	if key == "" {
		return false
	}
	for index, character := range key {
		if character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' {
			continue
		}
		if index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}
