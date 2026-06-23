package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	openHuntOutputDirKey = "OPENHUNT_OUTPUT_DIR"
	ollamaAPIURLKey      = "OLLAMA_API_URL"
	ollamaModelKey       = "OLLAMA_MODEL"

	defaultOllamaAPIURL = "http://localhost:11434"
	defaultOllamaModel  = "gemma4:e4b"
)

// Config contains the runtime configuration for The Forge.
type Config struct {
	OpenHuntOutputDir string
	OllamaAPIURL      string
	OllamaModel       string
}

// Load reads optional dotenv values and validates the runtime configuration.
// Values already present in the process environment take precedence.
func Load(dotenvPath string) (Config, error) {
	if err := loadDotEnv(dotenvPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load %s: %w", dotenvPath, err)
	}

	outputDir := strings.TrimSpace(os.Getenv(openHuntOutputDirKey))
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

	ollamaAPIURL := strings.TrimSpace(os.Getenv(ollamaAPIURLKey))
	if ollamaAPIURL == "" {
		ollamaAPIURL = defaultOllamaAPIURL
	}
	ollamaModel := strings.TrimSpace(os.Getenv(ollamaModelKey))
	if ollamaModel == "" {
		ollamaModel = defaultOllamaModel
	}

	return Config{
		OpenHuntOutputDir: absoluteDir,
		OllamaAPIURL:      ollamaAPIURL,
		OllamaModel:       ollamaModel,
	}, nil
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
