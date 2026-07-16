package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/admbahm/theForge/ckb/export"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
	"github.com/admbahm/theForge/ckb/planning"
	"github.com/admbahm/theForge/internal/config"
	"github.com/admbahm/theForge/internal/llm"
	"github.com/admbahm/theForge/pkg/engine"
	"github.com/admbahm/theForge/pkg/models"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "ckb" {
		handleCKBCommand(args[1:])
		return
	}

	// Parse CLI arguments using standard flag package
	flagSet := flag.NewFlagSet("theforge", flag.ExitOnError)
	tierFlag := flagSet.String("tier", "auto", "Funnel tier to run: local, frontier, auto")
	vaultFlag := flagSet.String("vault", "", "Path to the Obsidian vault / OpenHunt output directory")
	concurrencyFlag := flagSet.Int("concurrency", 0, "Number of concurrent workers")
	providerFlag := flagSet.String("provider", "", "LLM provider override")
	modelFlag := flagSet.String("model", "", "LLM model override")

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
	if ctx.Err() != nil {
		return nil
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("create LLM client: %w", err)
	}

	// For local or auto tiers, verify Ollama socket connectivity before starting
	if tier == "local" || tier == "auto" {
		if mm, ok := client.(llm.ModelManager); ok {
			if err := mm.Ping(ctx); err != nil {
				return fmt.Errorf("local Ollama server is unreachable: %w (please start Ollama or configure a remote provider)", err)
			}
		}
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

func handleCKBCommand(args []string) {
	if len(args) == 0 {
		log.Fatal("ckb command requires a subcommand: validate, export, or plan")
	}

	subCmd := args[0]
	switch subCmd {
	case "validate":
		runCKBValidate(args[1:])
	case "export":
		runCKBExport(args[1:])
	case "plan":
		runCKBPlan(args[1:])
	default:
		log.Fatalf("Unknown ckb subcommand %q. Supported: validate, export, plan", subCmd)
	}
}

func runCKBValidate(args []string) {
	fs := flag.NewFlagSet("ckb validate", flag.ExitOnError)
	dirFlag := fs.String("dir", "./ckb", "Path to the Career Knowledge Base directory")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}

	res := parser.ParseDirectory(context.Background(), *dirFlag, opts)

	hasErrors := false
	for _, diag := range res.Diagnostics {
		severity := "INFO"
		if diag.Severity == model.SeverityError {
			severity = "ERROR"
			hasErrors = true
		} else if diag.Severity == model.SeverityFatal {
			severity = "FATAL"
			hasErrors = true
		} else if diag.Severity == model.SeverityWarning {
			severity = "WARNING"
		}

		fmt.Printf("[%s] [%s] %s (File: %s, Line: %d)\n", severity, diag.Code, diag.Message, diag.Source.FilePath, diag.Source.Line)
	}

	if hasErrors {
		fmt.Println("Validation failed with errors.")
		os.Exit(1)
	}
	fmt.Println("Validation passed successfully!")
}

func runCKBExport(args []string) {
	fs := flag.NewFlagSet("ckb export", flag.ExitOnError)
	dirFlag := fs.String("dir", "./ckb", "Path to the Career Knowledge Base directory")
	formatFlag := fs.String("format", "json", "Output format: json")
	outFlag := fs.String("out", "", "Output file path (defaults to stdout)")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}

	res := parser.ParseDirectory(context.Background(), *dirFlag, opts)

	hasErrors := false
	for _, diag := range res.Diagnostics {
		if diag.Severity == model.SeverityError || diag.Severity == model.SeverityFatal {
			fmt.Printf("[ERROR] [%s] %s (File: %s, Line: %d)\n", diag.Code, diag.Message, diag.Source.FilePath, diag.Source.Line)
			hasErrors = true
		}
	}
	if hasErrors {
		log.Fatal("CKB contains errors. Export aborted.")
	}

	var w io.Writer = os.Stdout
	if *outFlag != "" {
		f, err := os.Create(*outFlag)
		if err != nil {
			log.Fatalf("failed to create output file: %v", err)
		}
		defer f.Close()
		w = f
	}

	switch strings.ToLower(*formatFlag) {
	case "json":
		if err := export.ExportJSON(res.KnowledgeBase, res.Diagnostics, w); err != nil {
			log.Fatalf("failed to export JSON: %v", err)
		}
	default:
		log.Fatalf("Unsupported export format %q. Supported: json", *formatFlag)
	}
}

func runCKBPlan(args []string) {
	fs := flag.NewFlagSet("ckb plan", flag.ExitOnError)
	jobFlag := fs.String("job", "", "Path to the JobPost markdown file")
	dirFlag := fs.String("dir", "./ckb", "Path to the Career Knowledge Base directory")
	outFlag := fs.String("out", "", "Output file path for the ArtifactPlan JSON (defaults to stdout)")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *jobFlag == "" {
		log.Fatal("-job path is required")
	}

	// 1. Read and parse JobPost
	jobData, err := os.ReadFile(*jobFlag)
	if err != nil {
		log.Fatalf("failed to read job file: %v", err)
	}

	var job models.JobPost
	if err := models.UnmarshalMarkdown(jobData, &job); err != nil {
		log.Fatalf("failed to parse job: %v", err)
	}

	// 2. Parse CKB
	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}
	ckbRes := parser.ParseDirectory(context.Background(), *dirFlag, opts)
	for _, diag := range ckbRes.Diagnostics {
		if diag.Severity == model.SeverityError || diag.Severity == model.SeverityFatal {
			log.Fatalf("CKB validation failed: %s (File: %s, Line: %d)", diag.Message, diag.Source.FilePath, diag.Source.Line)
		}
	}

	// 3. Assemble TargetProfile and PlanRequest
	target := &planning.TargetProfile{
		RoleTitle:           job.Title,
		DesiredTechnologies: job.TechStack,
	}

	req := planning.PlanRequest{
		ArtifactType: planning.TypeResume, // Default to resume
		PolicyID:     "StrictPublic",       // Default policy
		Target:       target,
	}

	// Calculate scores, gaps, budget, etc.
	planRes := planning.BuildPlan(context.Background(), ckbRes.KnowledgeBase, req)
	for _, diag := range planRes.Diagnostics {
		if diag.Severity == model.SeverityError || diag.Severity == model.SeverityFatal {
			log.Fatalf("Artifact planning failed: %s", diag.Message)
		}
	}

	var w io.Writer = os.Stdout
	if *outFlag != "" {
		f, err := os.Create(*outFlag)
		if err != nil {
			log.Fatalf("failed to create output file: %v", err)
		}
		defer f.Close()
		w = f
	}

	if err := export.ExportPlanJSON(planRes.Plan, w); err != nil {
		log.Fatalf("failed to export plan JSON: %v", err)
	}
}
