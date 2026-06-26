package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	DefaultLLMProvider  = "ollama"
	DefaultOllamaAPIURL = "http://localhost:11434"
	DefaultOllamaModel  = "gemma4:e4b"
	DefaultOpenAIKeyEnv = "OPENAI_API_KEY"
	DefaultOpenAIModel  = "gpt-4.1-mini"
	DefaultGeminiKeyEnv = "GEMINI_API_KEY"
	DefaultGeminiModel  = "gemini-2.5-flash"
	DefaultConcurrency  = 4

	concurrencyKey = "THEFORGE_CONCURRENCY"
)

// Config contains the runtime configuration for The Forge.
type Config struct {
	OpenHuntOutputDir string          `yaml:"openhunt_output_dir"`
	Concurrency       int             `yaml:"concurrency"`
	LLM               LLMConfig       `yaml:"llm"`
	Providers         ProvidersConfig `yaml:"providers"`

	// OllamaAPIURL and OllamaModel preserve the original programmatic config API.
	OllamaAPIURL string `yaml:"-"`
	OllamaModel  string `yaml:"-"`
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
	cfg.OllamaAPIURL = cfg.Providers.Ollama.Host
	cfg.OllamaModel = cfg.Providers.Ollama.Model
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Concurrency: DefaultConcurrency,
		LLM:         LLMConfig{Provider: DefaultLLMProvider},
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
	setFromEnvironment(&cfg.LLM.Provider, llmProviderKey)
	setFromEnvironment(&cfg.LLM.Model, llmModelKey)
	setFromEnvironment(&cfg.Providers.Ollama.Host, ollamaAPIURLKey)
	setFromEnvironment(&cfg.Providers.Ollama.Model, ollamaModelKey)
	setFromEnvironment(&cfg.Providers.OpenAI.APIKeyEnv, openAIAPIKeyEnvKey)
	setFromEnvironment(&cfg.Providers.OpenAI.Model, openAIModelKey)
	setFromEnvironment(&cfg.Providers.Gemini.APIKeyEnv, geminiAPIKeyEnvKey)
	setFromEnvironment(&cfg.Providers.Gemini.Model, geminiModelKey)
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
