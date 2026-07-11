package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverFiles walks the CKB directory recursively and returns sorted files.
func DiscoverFiles(root string) ([]string, error) {
	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return nil, fmt.Errorf("failed to clean root path: %w", err)
	}

	// Verify root directory exists
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return nil, fmt.Errorf("root directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", cleanRoot)
	}

	var discovered []string

	err = filepath.WalkDir(cleanRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Clean paths and calculate absolute representations
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		// Enforce path containment
		if !strings.HasPrefix(absPath, cleanRoot) {
			return fmt.Errorf("security boundary violation: path %s escaped root %s", absPath, cleanRoot)
		}

		// Check folder exclusions
		rel, err := filepath.Rel(cleanRoot, absPath)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")

		// Ignore hidden folders/files (starting with dot) or excluded folders (templates, examples, docs, tests)
		for _, part := range parts {
			if part == "." || part == ".." {
				continue
			}
			if strings.HasPrefix(part, ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			switch part {
			case "templates", "examples", "docs", "tests":
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Do not follow symlinks (walkDir does not follow symlinks by default, but double check type)
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		// File extension filter
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			base := filepath.Base(path)
			if base == "README.md" || base == "schema.md" || base == "metadata.md" || base == "walkthrough.md" {
				return nil
			}
			discovered = append(discovered, absPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort alphabetically by relative path to guarantee deterministic ordering
	sort.Slice(discovered, func(i, j int) bool {
		relI, _ := filepath.Rel(cleanRoot, discovered[i])
		relJ, _ := filepath.Rel(cleanRoot, discovered[j])
		return strings.ToLower(filepath.ToSlash(relI)) < strings.ToLower(filepath.ToSlash(relJ))
	})

	return discovered, nil
}
