package engine

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/admbahm/theForge/pkg/models"
	"github.com/fsnotify/fsnotify"
)

// Orchestrator monitors the Obsidian vault and processes job posts.
type Orchestrator struct {
	vaultPath string
	watcher   *fsnotify.Watcher
	stopChan  chan struct{}
}

// NewOrchestrator creates a new Orchestrator instance.
func NewOrchestrator(vaultPath string) (*Orchestrator, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &Orchestrator{
		vaultPath: vaultPath,
		watcher:   watcher,
		stopChan:  make(chan struct{}),
	}, nil
}

// Start begins the vault monitoring and performs an initial scan.
func (o *Orchestrator) Start() error {
	// 1. Initial batch processing fallback
	log.Printf("Starting initial vault scan: %s", o.vaultPath)
	if err := o.processVault(); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	// 2. Start non-blocking watcher
	if err := o.watcher.Add(o.vaultPath); err != nil {
		return fmt.Errorf("failed to add vault path to watcher: %w", err)
	}

	go o.watch()

	return nil
}

// Stop stops the orchestrator.
func (o *Orchestrator) Stop() {
	close(o.stopChan)
	o.watcher.Close()
}

// watch listens for filesystem events.
func (o *Orchestrator) watch() {
	for {
		select {
		case event, ok := <-o.watcher.Events:
			if !ok {
				return
			}
			// Only care about creation and writes for .md files
			if strings.HasSuffix(event.Name, ".md") {
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					log.Printf("File event detected: %s (%s)", event.Name, event.Op)
					o.handleFile(event.Name)
				}
			}
		case err, ok := <-o.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		case <-o.stopChan:
			return
		}
	}
}

// processVault performs a sequential crawl of the vault directory.
func (o *Orchestrator) processVault() error {
	return filepath.Walk(o.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			o.handleFile(path)
		}
		return nil
	})
}

// handleFile processes a single Markdown file.
func (o *Orchestrator) handleFile(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file %s: %v", path, err)
		return
	}

	var jp models.JobPost
	err = models.UnmarshalMarkdown(data, &jp)
	if err != nil {
		// Might not be a valid job post file, skip silently or log at debug level
		return
	}

	// Check if state == "favorite" or favorite == true
	if jp.State == "favorite" || jp.Favorite {
		if jp.State == "intel-ready" {
			// Already processed
			return
		}

		fmt.Printf("MOCK: Processing Phase 2 Intel for [%s] - [%s]\n", jp.Company, jp.Title)

		// Programmatically update state
		jp.State = "intel-ready"

		// Save back to disk
		updatedData, err := jp.MarshalMarkdown()
		if err != nil {
			log.Printf("Error marshaling job post %s: %v", path, err)
			return
		}

		err = ioutil.WriteFile(path, updatedData, 0644)
		if err != nil {
			log.Printf("Error writing file %s: %v", path, err)
			return
		}
		log.Printf("Updated state to 'intel-ready' for %s", path)
	}
}
