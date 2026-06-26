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

// Orchestrator monitors the Obsidian vault and processes job posts.
type Orchestrator struct {
	vaultPath string
	generator IntelGenerator
	watcher   *fsnotify.Watcher
	ctx       context.Context
	cancel    context.CancelFunc
	stopOnce  sync.Once
	jobs      chan string
	pendingMu sync.Mutex
	pending   map[string]struct{}
	workers   sync.WaitGroup
}

const jobQueueSize = 32

// NewOrchestrator creates a new Orchestrator instance.
func NewOrchestrator(vaultPath string, generator IntelGenerator) (*Orchestrator, error) {
	if generator == nil {
		return nil, fmt.Errorf("intel generator is required")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &Orchestrator{
		vaultPath: vaultPath,
		generator: generator,
		watcher:   watcher,
		ctx:       ctx,
		cancel:    cancel,
		jobs:      make(chan string, jobQueueSize),
		pending:   make(map[string]struct{}),
	}, nil
}

// Start begins recursive vault monitoring and performs an initial scan.
func (o *Orchestrator) Start() error {
	if err := o.addWatches(o.vaultPath); err != nil {
		return fmt.Errorf("add vault watches: %w", err)
	}
	o.workers.Add(1)
	go o.processJobs()

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
		log.Printf("Error reading file %s: %v", path, err)
		return
	}

	var job models.JobPost
	if err := models.UnmarshalMarkdown(data, &job); err != nil {
		return
	}
	if job.State != "favorite" && !(job.State == "" && job.Favorite) {
		return
	}

	log.Printf("Generating intelligence for [%s] - [%s]", job.Company, job.Title)
	intel, err := o.generator.GenerateIntel(o.ctx, job)
	if err != nil {
		log.Printf("Error generating intelligence for %s: %v", path, err)
		return
	}

	updatedData, err := models.UpdateStateAndAppendIntel(data, "intel-ready", intel)
	if err != nil {
		log.Printf("Error updating job post %s: %v", path, err)
		return
	}
	if err := atomicWrite(path, updatedData); err != nil {
		log.Printf("Error writing job post %s: %v", path, err)
		return
	}
	log.Printf("Generated intelligence and updated state to 'intel-ready' for %s", path)
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
