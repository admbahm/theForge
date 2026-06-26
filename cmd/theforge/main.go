package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/admbahm/theForge/internal/config"
	"github.com/admbahm/theForge/internal/llm"
	"github.com/admbahm/theForge/pkg/engine"
)

func main() {
	cfg, err := config.Load(".env", "theforge.yaml")
	if err != nil {
		log.Fatal(err)
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if err := run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cfg config.Config) error {
	client, err := llm.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("create LLM client: %w", err)
	}

	orchestrator, err := engine.NewOrchestratorWithConcurrency(cfg.OpenHuntOutputDir, client, cfg.Concurrency)
	if err != nil {
		return fmt.Errorf("create orchestrator: %w", err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.Start(); err != nil {
		return fmt.Errorf("start orchestrator: %w", err)
	}

	provider := cfg.LLM.Provider
	if provider == "" {
		provider = config.DefaultLLMProvider
	}
	log.Printf("Watching OpenHunt output directory: %s (LLM provider: %s)", cfg.OpenHuntOutputDir, provider)
	<-ctx.Done()
	log.Printf("Shutdown requested")
	return nil
}
