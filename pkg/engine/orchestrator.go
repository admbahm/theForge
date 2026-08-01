package engine

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/admbahm/theForge/ckb/export"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
	"github.com/admbahm/theForge/ckb/planning"
	"github.com/admbahm/theForge/ckb/rendering"
	"github.com/admbahm/theForge/pkg/models"
	"github.com/fsnotify/fsnotify"
)

// IntelGenerator produces Markdown intelligence for a job posting.
type IntelGenerator interface {
	GenerateIntel(context.Context, models.JobPost) (string, error)
}

type Orchestrator struct {
	vaultPath   string
	generator   IntelGenerator
	concurrency int
	watcher     *fsnotify.Watcher
	ctx         context.Context
	cancel      context.CancelFunc
	stopOnce    sync.Once
	jobs        chan string
	pendingMu   sync.Mutex
	// pending tracks queued or active paths. A true value means another event
	// arrived while the path was pending and requires one follow-up pass.
	pending     map[string]bool
	workers     sync.WaitGroup
	tier        string
	application ApplicationConfig
	publisher   packetPublisher
}

type ApplicationConfig struct {
	CKBDir   string
	DemoMode bool
	Contact  rendering.ContactInfo
}

const jobQueueSize = 32

// NewOrchestrator creates a new Orchestrator instance with the default concurrency level.
func NewOrchestrator(vaultPath string, generator IntelGenerator) (*Orchestrator, error) {
	return NewOrchestratorWithConcurrency(vaultPath, generator, 4)
}

// NewOrchestratorWithConcurrency creates a new Orchestrator instance with a custom concurrency limit.
func NewOrchestratorWithConcurrency(vaultPath string, generator IntelGenerator, concurrency int) (*Orchestrator, error) {
	if generator == nil {
		return nil, fmt.Errorf("intel generator is required")
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &Orchestrator{
		vaultPath:   vaultPath,
		generator:   generator,
		concurrency: concurrency,
		watcher:     watcher,
		ctx:         ctx,
		cancel:      cancel,
		jobs:        make(chan string, jobQueueSize),
		pending:     make(map[string]bool),
		tier:        "auto",
		publisher:   defaultPacketPublisher(),
	}, nil
}

// SetTier configures the orchestrator's run tier: local, frontier, or auto.
func (o *Orchestrator) SetTier(tier string) error {
	tier = strings.ToLower(strings.TrimSpace(tier))
	if tier != "local" && tier != "frontier" && tier != "auto" {
		return fmt.Errorf("invalid tier %q (supported: local, frontier, auto)", tier)
	}
	o.tier = tier
	return nil
}

// SetApplicationConfig supplies the explicitly configured evidence and identity
// used for public application artifacts. Validation is repeated at processing
// time so an unsafe configuration can never silently fall back to examples.
func (o *Orchestrator) SetApplicationConfig(cfg ApplicationConfig) {
	o.application = cfg
}

// Start begins recursive vault monitoring and performs an initial scan.
func (o *Orchestrator) Start() error {
	if err := o.addWatches(o.vaultPath); err != nil {
		return fmt.Errorf("add vault watches: %w", err)
	}
	for i := 0; i < o.concurrency; i++ {
		o.workers.Add(1)
		go o.processJobs()
	}

	log.Printf("Starting initial vault scan: %s", o.vaultPath)
	if err := o.processVault(); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	o.workers.Add(1)
	go o.watch()
	return nil
}

// Stop stops the orchestrator. It is safe to call more than once.
func (o *Orchestrator) Stop() {
	o.stopOnce.Do(func() {
		o.cancel()
		if err := o.watcher.Close(); err != nil {
			log.Printf("Error closing watcher: %v", err)
		}
		o.workers.Wait()
	})
}

func (o *Orchestrator) watch() {
	defer o.workers.Done()
	for {
		select {
		case event, ok := <-o.watcher.Events:
			if !ok {
				return
			}
			if o.isApplicationOutput(event.Name) {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := o.addWatches(event.Name); err != nil {
						log.Printf("Error watching new directory %s: %v", event.Name, err)
					}
					continue
				}
			}
			if strings.EqualFold(filepath.Ext(event.Name), ".md") &&
				event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				log.Printf("File event detected: %s (%s)", event.Name, event.Op)
				o.enqueue(event.Name)
			}
		case err, ok := <-o.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		case <-o.ctx.Done():
			return
		}
	}
}

func (o *Orchestrator) addWatches(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && o.isApplicationOutput(path) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			if err := o.watcher.Add(path); err != nil {
				return fmt.Errorf("watch %s: %w", path, err)
			}
		}
		return nil
	})
}

func (o *Orchestrator) processVault() error {
	return filepath.WalkDir(o.vaultPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && o.isApplicationOutput(path) {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			o.enqueue(path)
		}
		return nil
	})
}

func (o *Orchestrator) enqueue(path string) {
	path = filepath.Clean(path)
	if o.isApplicationOutput(path) {
		return
	}
	o.pendingMu.Lock()
	if _, exists := o.pending[path]; exists {
		o.pending[path] = true
		o.pendingMu.Unlock()
		return
	}
	o.pending[path] = false
	o.pendingMu.Unlock()

	select {
	case o.jobs <- path:
	case <-o.ctx.Done():
		o.clearPending(path)
	}
}

func (o *Orchestrator) isApplicationOutput(path string) bool {
	outputRoot := filepath.Join(filepath.Clean(o.vaultPath), "applications")
	relative, err := filepath.Rel(outputRoot, filepath.Clean(path))
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (o *Orchestrator) processJobs() {
	defer o.workers.Done()
	for {
		select {
		case path := <-o.jobs:
			o.handleFile(path)
			o.finishPending(path)
		case <-o.ctx.Done():
			return
		}
	}
}

func (o *Orchestrator) clearPending(path string) {
	o.pendingMu.Lock()
	delete(o.pending, path)
	o.pendingMu.Unlock()
}

func (o *Orchestrator) finishPending(path string) {
	o.pendingMu.Lock()
	dirty, exists := o.pending[path]
	if !exists || !dirty {
		delete(o.pending, path)
		o.pendingMu.Unlock()
		return
	}
	o.pending[path] = false
	o.pendingMu.Unlock()

	select {
	case o.jobs <- path:
	case <-o.ctx.Done():
		o.clearPending(path)
	}
}

func (o *Orchestrator) handleFile(path string) {
	fileName := filepath.Base(path)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[Error] Failed to read job post file %s: %v", path, err)
		return
	}

	var job models.JobPost
	if err := models.UnmarshalMarkdown(data, &job); err != nil {
		log.Printf("[Error] [%s] Failed to parse job post: %v", fileName, err)
		return
	}
	if job.Company == "" {
		return
	}

	if job.State == "apply" {
		log.Printf("[Processing] [%s] %s - %s: Processing application artifacts...", fileName, job.Company, job.Title)
		if err := o.processApply(path, job); err != nil {
			log.Printf("[Error] [%s] %s - %s: Application generation failed: %v", fileName, job.Company, job.Title, err)
			return
		}
		updatedData, err := models.UpdateStateOnly(data, "completed")
		if err != nil {
			log.Printf("[Error] [%s] %s - %s: Failed to update note state: %v", fileName, job.Company, job.Title, err)
			return
		}
		if err := atomicWrite(path, updatedData); err != nil {
			log.Printf("[Error] [%s] %s - %s: Failed to save changes: %v", fileName, job.Company, job.Title, err)
			return
		}
		log.Printf("[Success] [%s] %s - %s: Finished application generation (Status: completed)", fileName, job.Company, job.Title)
		return
	}

	tier := o.tier
	if tier == "" {
		tier = "auto"
	}

	var targetState string
	var processingTier string

	switch tier {
	case "local":
		if job.State == "new" || (job.State == "" && !job.Favorite) {
			targetState = "processed"
			processingTier = "local"
		}
	case "frontier":
		if job.State == "favorite" || (job.State == "" && job.Favorite) {
			targetState = "intel-ready"
			processingTier = "frontier"
		}
	case "auto":
		if job.State == "new" || (job.State == "" && !job.Favorite) {
			targetState = "processed"
			processingTier = "local"
		} else if job.State == "favorite" || (job.State == "" && job.Favorite) {
			targetState = "intel-ready"
			processingTier = "frontier"
		}
	}

	if targetState == "" {
		return
	}

	log.Printf("[Processing] [%s] %s - %s: Generating %s intelligence...", fileName, job.Company, job.Title, processingTier)
	runCtx := context.WithValue(o.ctx, "tier", processingTier)

	if optimizer, ok := o.generator.(interface {
		OptimizeVRAM(ctx context.Context, targetModel string) error
	}); ok {
		if err := optimizer.OptimizeVRAM(runCtx, ""); err != nil {
			log.Printf("[Warning] VRAM optimization failed: %v", err)
		}
	}

	intel, err := o.generator.GenerateIntel(runCtx, job)
	if err != nil {
		log.Printf("[Error] [%s] %s - %s: Intel generation failed: %v", fileName, job.Company, job.Title, err)
		return
	}

	conf := models.ComputeConfidence(job)
	updatedData, err := models.UpdateStateAndAppendIntel(data, targetState, intel, conf)
	if err != nil {
		log.Printf("[Error] [%s] %s - %s: Failed to update note payload: %v", fileName, job.Company, job.Title, err)
		return
	}
	if err := atomicWrite(path, updatedData); err != nil {
		log.Printf("[Error] [%s] %s - %s: Failed to save changes: %v", fileName, job.Company, job.Title, err)
		return
	}
	log.Printf("[Success] [%s] %s - %s: Finished intelligence (Status: %s)", fileName, job.Company, job.Title, targetState)
}

func (o *Orchestrator) processApply(path string, job models.JobPost) error {
	ckbPath := strings.TrimSpace(o.application.CKBDir)
	if ckbPath == "" {
		return fmt.Errorf("application configuration: THEFORGE_CKB_DIR is required; no artifacts were published")
	}
	if strings.TrimSpace(o.application.Contact.Name) == "" {
		return fmt.Errorf("application configuration: THEFORGE_CONTACT_NAME is required; no artifacts were published")
	}
	if strings.TrimSpace(o.application.Contact.Email) == "" {
		return fmt.Errorf("application configuration: THEFORGE_CONTACT_EMAIL is required; no artifacts were published")
	}

	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}

	ckbRes := parser.ParseDirectory(o.ctx, ckbPath, opts)

	if model.HasBlockingDiagnostics(ckbRes.Diagnostics) {
		return blockingDiagnosticError("CKB parsing", ckbRes.Diagnostics)
	}

	target := &planning.TargetProfile{
		RoleTitle:           job.Title,
		Company:             job.Company,
		DesiredTechnologies: job.TechStack,
	}

	resumeReq := planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     "StrictPublic",
		Target:       target,
	}
	resumePlanRes := planning.BuildPlan(o.ctx, ckbRes.KnowledgeBase, resumeReq)
	if model.HasBlockingDiagnostics(resumePlanRes.Diagnostics) {
		return blockingDiagnosticError("resume planning", resumePlanRes.Diagnostics)
	}
	if resumePlanRes.Plan == nil {
		return fmt.Errorf("resume plan construction failed")
	}

	clReq := planning.PlanRequest{
		ArtifactType: planning.TypeCoverLetter,
		PolicyID:     "StrictPublic",
		Target:       target,
	}
	clPlanRes := planning.BuildPlan(o.ctx, ckbRes.KnowledgeBase, clReq)
	if model.HasBlockingDiagnostics(clPlanRes.Diagnostics) {
		return blockingDiagnosticError("cover letter planning", clPlanRes.Diagnostics)
	}
	if clPlanRes.Plan == nil {
		return fmt.Errorf("cover letter plan construction failed")
	}

	contact := o.application.Contact

	resumeRenderReq := rendering.RenderRequest{
		Plan:    resumePlanRes.Plan,
		Options: rendering.RenderOptions{Contact: contact},
	}
	resumeRenderRes := rendering.Render(o.ctx, resumeRenderReq)
	if model.HasBlockingDiagnostics(resumeRenderRes.Diagnostics) {
		return blockingDiagnosticError("resume rendering", resumeRenderRes.Diagnostics)
	}
	if resumeRenderRes.Artifact == nil {
		return fmt.Errorf("resume rendering failed")
	}

	clRenderReq := rendering.RenderRequest{
		Plan:    clPlanRes.Plan,
		Options: rendering.RenderOptions{Contact: contact},
	}
	clRenderRes := rendering.Render(o.ctx, clRenderReq)
	if model.HasBlockingDiagnostics(clRenderRes.Diagnostics) {
		return blockingDiagnosticError("cover letter rendering", clRenderRes.Diagnostics)
	}
	if clRenderRes.Artifact == nil {
		return fmt.Errorf("cover letter rendering failed")
	}

	resumeBytes, err := renderArtifactMarkdown(resumeRenderRes.Artifact, o.application.DemoMode)
	if err != nil {
		return fmt.Errorf("export resume markdown: %w", err)
	}
	coverLetterBytes, err := renderArtifactMarkdown(clRenderRes.Artifact, o.application.DemoMode)
	if err != nil {
		return fmt.Errorf("export cover letter markdown: %w", err)
	}

	packetName := applicationPacketName(o.vaultPath, path, job)
	applicationsDir := filepath.Join(o.vaultPath, "applications")
	packet := applicationPacket{
		Files: map[string][]byte{
			"cover_letter.md": coverLetterBytes,
			"resume.md":       resumeBytes,
		},
		Manifest: packetManifest{
			SchemaVersion:   packetSchemaVersion,
			CompilerVersion: "theforge-phase3-alpha",
			Job: packetJobIdentity{
				JobID:   job.JobID,
				Company: job.Company,
				Title:   job.Title,
			},
			DemoMode: o.application.DemoMode,
			Warnings: mergeWarningCodes(resumePlanRes.Diagnostics, clPlanRes.Diagnostics, resumeRenderRes.Diagnostics, clRenderRes.Diagnostics),
			Files: []packetManifestFile{
				manifestFileForArtifact("cover_letter.md", coverLetterBytes, clRenderRes.Artifact),
				manifestFileForArtifact("resume.md", resumeBytes, resumeRenderRes.Artifact),
			},
		},
	}
	if err := o.publisher.publish(applicationsDir, packetName, packet); err != nil {
		return fmt.Errorf("publish application packet: %w", err)
	}

	log.Printf("[Success] Tailored application packet generated at %s", filepath.Join(applicationsDir, packetName))
	return nil
}

func renderArtifactMarkdown(artifact *rendering.Artifact, demoMode bool) ([]byte, error) {
	var output bytes.Buffer
	if demoMode {
		output.WriteString("> **THE FORGE DEMO OUTPUT — FICTIONAL DATA — DO NOT SUBMIT**\n\n")
	}
	if err := export.ExportMarkdown(artifact, &output, export.MarkdownOptions{IncludeHeadings: true}); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func manifestFileForArtifact(name string, data []byte, artifact *rendering.Artifact) packetManifestFile {
	return packetManifestFile{
		Name:               name,
		ArtifactType:       string(artifact.Type),
		SHA256:             digestBytes(data),
		Size:               len(data),
		ContentDigest:      artifact.Manifest.ContentDigest,
		SourceReferences:   append([]string(nil), artifact.Manifest.SourceReferences...),
		EvidenceReferences: append([]string(nil), artifact.Manifest.EvidenceReferences...),
		Warnings:           append([]string(nil), artifact.Manifest.Warnings...),
	}
}

func mergeWarningCodes(groups ...[]model.Diagnostic) []string {
	seen := make(map[string]struct{})
	var warnings []string
	for _, diagnostics := range groups {
		for _, diagnostic := range diagnostics {
			if diagnostic.Severity != model.SeverityWarning {
				continue
			}
			code := string(diagnostic.Code)
			if _, exists := seen[code]; exists {
				continue
			}
			seen[code] = struct{}{}
			warnings = append(warnings, code)
		}
	}
	sort.Strings(warnings)
	return warnings
}

func blockingDiagnosticError(stage string, diagnostics []model.Diagnostic) error {
	codes := model.BlockingDiagnosticCodes(diagnostics)
	formatted := make([]string, len(codes))
	for index, code := range codes {
		formatted[index] = string(code)
	}
	return fmt.Errorf("%s blocked by diagnostic code(s): %s; correct the source data before retrying", stage, strings.Join(formatted, ", "))
}

func sanitizePathSegment(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return strings.ToLower(s)
}

func applicationPacketName(vaultPath, sourcePath string, job models.JobPost) string {
	relativePath, err := filepath.Rel(filepath.Clean(vaultPath), filepath.Clean(sourcePath))
	if err != nil {
		relativePath = filepath.Clean(sourcePath)
	}
	identity := job.JobID + "\x00" + filepath.ToSlash(relativePath)
	suffix := digestBytes([]byte(identity))[:12]
	return fmt.Sprintf("%s-%s-%s", sanitizePathSegment(job.Company), sanitizePathSegment(job.Title), suffix)
}

func atomicWrite(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := file.Chmod(info.Mode().Perm()); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
