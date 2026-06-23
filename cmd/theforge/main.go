package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/admbahm/theForge/internal/config"
	"github.com/admbahm/theForge/internal/ollama"
	"github.com/admbahm/theForge/pkg/engine"
)

func main() {
	cfg, err := config.Load(".env")
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
	ollamaClient, err := ollama.NewClient(cfg.OllamaAPIURL, cfg.OllamaModel)
	if err != nil {
		return fmt.Errorf("create Ollama client: %w", err)
	}

	orchestrator, err := engine.NewOrchestrator(cfg.OpenHuntOutputDir, ollamaClient)
	if err != nil {
		return fmt.Errorf("create orchestrator: %w", err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.Start(); err != nil {
		return fmt.Errorf("start orchestrator: %w", err)
	}

	log.Printf("Watching OpenHunt output directory: %s (Ollama model: %s)", cfg.OpenHuntOutputDir, cfg.OllamaModel)
	<-ctx.Done()
	log.Printf("Shutdown requested")
	return nil
}
