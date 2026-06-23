package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsDotEnvAndValidatesDirectory(t *testing.T) {
	outputDir := t.TempDir()
	dotenvPath := filepath.Join(t.TempDir(), ".env")
	content := `OPENHUNT_OUTPUT_DIR="` + outputDir + `"` + "\n"
	if err := os.WriteFile(dotenvPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(openHuntOutputDirKey, "")
	if err := os.Unsetenv(openHuntOutputDirKey); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dotenvPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OpenHuntOutputDir != outputDir {
		t.Fatalf("OpenHuntOutputDir = %q, want %q", cfg.OpenHuntOutputDir, outputDir)
	}
}

func TestLoadEnvironmentOverridesDotEnv(t *testing.T) {
	environmentDir := t.TempDir()
	dotenvDir := t.TempDir()
	dotenvPath := filepath.Join(t.TempDir(), ".env")
	content := "OPENHUNT_OUTPUT_DIR=" + dotenvDir + "\n"
	if err := os.WriteFile(dotenvPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(openHuntOutputDirKey, environmentDir)

	cfg, err := Load(dotenvPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OpenHuntOutputDir != environmentDir {
		t.Fatalf("OpenHuntOutputDir = %q, want %q", cfg.OpenHuntOutputDir, environmentDir)
	}
}

func TestLoadRejectsMissingDirectory(t *testing.T) {
	t.Setenv(openHuntOutputDirKey, filepath.Join(t.TempDir(), "missing"))

	_, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err == nil || !strings.Contains(err.Error(), "validate OPENHUNT_OUTPUT_DIR") {
		t.Fatalf("Load() error = %v, want directory validation error", err)
	}
}

func TestParseDotEnvLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantOK    bool
		wantError bool
	}{
		{name: "comment", line: " # ignored", wantOK: false},
		{name: "export", line: `export NAME="value with spaces"`, wantKey: "NAME", wantValue: "value with spaces", wantOK: true},
		{name: "single quoted", line: "NAME='literal value'", wantKey: "NAME", wantValue: "literal value", wantOK: true},
		{name: "invalid key", line: "1NAME=value", wantError: true},
		{name: "missing separator", line: "NAME", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key, value, ok, err := parseDotEnvLine(test.line)
			if (err != nil) != test.wantError {
				t.Fatalf("parseDotEnvLine() error = %v, wantError %v", err, test.wantError)
			}
			if key != test.wantKey || value != test.wantValue || ok != test.wantOK {
				t.Fatalf("parseDotEnvLine() = (%q, %q, %v), want (%q, %q, %v)", key, value, ok, test.wantKey, test.wantValue, test.wantOK)
			}
		})
	}
}
