package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/admbahm/theForge/internal/config"
	"github.com/admbahm/theForge/internal/llm"
	"github.com/admbahm/theForge/pkg/engine"
)

func main() {
	// Parse CLI arguments using standard flag package
	flagSet := flag.NewFlagSet("theforge", flag.ExitOnError)
	tierFlag := flagSet.String("tier", "auto", "Funnel tier to run: local, frontier, auto")
	vaultFlag := flagSet.String("vault", "", "Path to the Obsidian vault / OpenHunt output directory")
	concurrencyFlag := flagSet.Int("concurrency", 0, "Number of concurrent workers")
	providerFlag := flagSet.String("provider", "", "LLM provider override")
	modelFlag := flagSet.String("model", "", "LLM model override")

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "run" {
		args = args[1:]
	}
	if err := flagSet.Parse(args); err != nil {
		log.Fatal(err)
	}

	tier := strings.ToLower(strings.TrimSpace(*tierFlag))
	if tier != "local" && tier != "frontier" && tier != "auto" {
		log.Fatalf("Invalid tier %q. Allowed values: local, frontier, auto", *tierFlag)
	}

	cfg, err := config.Load(".env", "theforge.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Apply CLI overrides to configuration
	if *vaultFlag != "" {
		cfg.OpenHuntOutputDir = *vaultFlag
	}
	if *concurrencyFlag > 0 {
		cfg.Concurrency = *concurrencyFlag
	}
	if *providerFlag != "" {
		cfg.LLM.Provider = *providerFlag
	}
	if *modelFlag != "" {
		cfg.LLM.Model = *modelFlag
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if err := run(ctx, cfg, tier); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cfg config.Config, tier string) error {
	client, err := llm.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("create LLM client: %w", err)
	}

	orchestrator, err := engine.NewOrchestratorWithConcurrency(cfg.OpenHuntOutputDir, client, cfg.Concurrency)
	if err != nil {
		return fmt.Errorf("create orchestrator: %w", err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.SetTier(tier); err != nil {
		return fmt.Errorf("set orchestrator tier: %w", err)
	}

	if err := orchestrator.Start(); err != nil {
		return fmt.Errorf("start orchestrator: %w", err)
	}

	provider := cfg.LLM.Provider
	if provider == "" {
		provider = config.DefaultLLMProvider
	}
	log.Printf("Watching OpenHunt output directory: %s (LLM provider: %s, Tier: %s)", cfg.OpenHuntOutputDir, provider, tier)
	<-ctx.Done()
	log.Printf("Shutdown requested")
	return nil
}
