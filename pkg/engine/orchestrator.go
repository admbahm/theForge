package engine

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	pending     map[string]struct{}
	workers     sync.WaitGroup
	tier        string
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
		pending:     make(map[string]struct{}),
		tier:        "auto",
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
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			o.enqueue(path)
		}
		return nil
	})
}

func (o *Orchestrator) enqueue(path string) {
	path = filepath.Clean(path)
	o.pendingMu.Lock()
	if _, exists := o.pending[path]; exists {
		o.pendingMu.Unlock()
		return
	}
	o.pending[path] = struct{}{}
	o.pendingMu.Unlock()

	select {
	case o.jobs <- path:
	case <-o.ctx.Done():
		o.clearPending(path)
	}
}

func (o *Orchestrator) processJobs() {
	defer o.workers.Done()
	for {
		select {
		case path := <-o.jobs:
			o.handleFile(path)
			o.clearPending(path)
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

func (o *Orchestrator) handleFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[Error] Failed to read job post file %s: %v", path, err)
		return
	}

	var job models.JobPost
	if err := models.UnmarshalMarkdown(data, &job); err != nil {
		return
	}
	if job.Company == "" {
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

	fileName := filepath.Base(path)
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

	updatedData, err := models.UpdateStateAndAppendIntel(data, targetState, intel)
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
