package main

import (
	"context"
	"testing"

	"github.com/admbahm/theForge/internal/config"
)

func TestRunStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := run(ctx, config.Config{
		OpenHuntOutputDir: t.TempDir(),
		OllamaAPIURL:      "http://localhost:11434",
		OllamaModel:       "gemma4:e4b",
	}, "auto")
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
}
