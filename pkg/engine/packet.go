package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const packetSchemaVersion = "1.0"

type applicationPacket struct {
	Manifest packetManifest
	Files    map[string][]byte
}

type packetManifest struct {
	SchemaVersion   string               `json:"schema_version"`
	CompilerVersion string               `json:"compiler_version"`
	Job             packetJobIdentity    `json:"job"`
	DemoMode        bool                 `json:"demo_mode"`
	Warnings        []string             `json:"warnings,omitempty"`
	Files           []packetManifestFile `json:"files"`
}

type packetJobIdentity struct {
	JobID   string `json:"job_id,omitempty"`
	Company string `json:"company"`
	Title   string `json:"title"`
}

type packetManifestFile struct {
	Name               string   `json:"name"`
	ArtifactType       string   `json:"artifact_type"`
	SHA256             string   `json:"sha256"`
	Size               int      `json:"size"`
	ContentDigest      string   `json:"content_digest"`
	SourceReferences   []string `json:"source_references"`
	EvidenceReferences []string `json:"evidence_references,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

type packetPublisher struct {
	rename    func(string, string) error
	removeAll func(string) error
	syncDir   func(string) error
	writeFile func(string, []byte, os.FileMode) error
}

func defaultPacketPublisher() packetPublisher {
	return packetPublisher{
		rename:    os.Rename,
		removeAll: os.RemoveAll,
		syncDir:   syncDirectory,
		writeFile: writeSyncedFile,
	}
}

func (p packetPublisher) publish(parentDir, packetName string, packet applicationPacket) error {
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return fmt.Errorf("create application packet parent: %w", err)
	}
	stageDir, err := os.MkdirTemp(parentDir, "."+packetName+".stage-")
	if err != nil {
		return fmt.Errorf("create application packet staging directory: %w", err)
	}
	if err := os.Chmod(stageDir, 0o700); err != nil {
		p.removeAll(stageDir)
		return fmt.Errorf("secure application packet staging directory: %w", err)
	}
	stageOwned := true
	defer func() {
		if stageOwned {
			_ = p.removeAll(stageDir)
		}
	}()

	names := make([]string, 0, len(packet.Files))
	for name := range packet.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if filepath.Base(name) != name {
			return fmt.Errorf("invalid packet filename %q", name)
		}
		if err := p.writeFile(filepath.Join(stageDir, name), packet.Files[name], 0o600); err != nil {
			return fmt.Errorf("stage packet file %s: %w", name, err)
		}
	}

	manifestBytes, err := json.MarshalIndent(packet.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode packet manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := p.writeFile(filepath.Join(stageDir, "manifest.json"), manifestBytes, 0o600); err != nil {
		return fmt.Errorf("stage packet manifest: %w", err)
	}
	if err := p.syncDir(stageDir); err != nil {
		return fmt.Errorf("sync packet staging directory: %w", err)
	}

	finalDir := filepath.Join(parentDir, packetName)
	backupDir := filepath.Join(parentDir, fmt.Sprintf(".%s.backup-%d", packetName, time.Now().UnixNano()))
	finalExists := false
	if info, statErr := os.Stat(finalDir); statErr == nil {
		if !info.IsDir() {
			return fmt.Errorf("packet destination %q is not a directory", finalDir)
		}
		finalExists = true
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect packet destination: %w", statErr)
	}

	if finalExists {
		if err := p.rename(finalDir, backupDir); err != nil {
			return fmt.Errorf("preserve previous packet: %w", err)
		}
	}
	if err := p.rename(stageDir, finalDir); err != nil {
		if finalExists {
			if rollbackErr := p.rename(backupDir, finalDir); rollbackErr != nil {
				return fmt.Errorf("publish packet: %w (rollback failed: %v)", err, rollbackErr)
			}
		}
		return fmt.Errorf("publish packet: %w", err)
	}
	stageOwned = false
	if err := p.syncDir(parentDir); err != nil {
		return fmt.Errorf("sync published packet parent: %w", err)
	}
	if finalExists {
		if err := p.removeAll(backupDir); err != nil {
			return fmt.Errorf("remove previous packet backup: %w", err)
		}
		if err := p.syncDir(parentDir); err != nil {
			return fmt.Errorf("sync packet backup removal: %w", err)
		}
	}
	return nil
}

func writeSyncedFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
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
	return file.Close()
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
