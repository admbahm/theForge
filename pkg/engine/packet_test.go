package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPacketPublisherPublishesCompletePacketWithManifest(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "applications")
	files := map[string][]byte{
		"cover_letter.md": []byte("cover letter\n"),
		"resume.md":       []byte("resume\n"),
	}
	manifest := packetManifest{
		SchemaVersion:   packetSchemaVersion,
		CompilerVersion: "test",
		Job:             packetJobIdentity{JobID: "R123", Company: "Example", Title: "Engineer"},
		Files: []packetManifestFile{
			{Name: "cover_letter.md", SHA256: digestBytes(files["cover_letter.md"]), Size: len(files["cover_letter.md"])},
			{Name: "resume.md", SHA256: digestBytes(files["resume.md"]), Size: len(files["resume.md"])},
		},
	}
	if err := defaultPacketPublisher().publish(parent, "example-engineer", applicationPacket{Manifest: manifest, Files: files}); err != nil {
		t.Fatal(err)
	}

	packetDir := filepath.Join(parent, "example-engineer")
	for name, expected := range files {
		path := filepath.Join(packetDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(expected) {
			t.Fatalf("%s = %q, want %q", name, data, expected)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s permissions = %o, want 600", name, info.Mode().Perm())
		}
	}

	manifestData, err := os.ReadFile(filepath.Join(packetDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var published packetManifest
	if err := json.Unmarshal(manifestData, &published); err != nil {
		t.Fatal(err)
	}
	for _, file := range published.Files {
		data, err := os.ReadFile(filepath.Join(packetDir, file.Name))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		if file.SHA256 != hex.EncodeToString(digest[:]) || file.Size != len(data) {
			t.Fatalf("manifest mismatch for %s: %+v", file.Name, file)
		}
	}
	info, err := os.Stat(packetDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("packet directory permissions = %o, want 700", info.Mode().Perm())
	}
}

func TestPacketPublisherRollsBackFailedReplacement(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "applications")
	packetName := "example-engineer"
	oldPacket := applicationPacket{
		Manifest: packetManifest{SchemaVersion: packetSchemaVersion},
		Files:    map[string][]byte{"resume.md": []byte("previous valid packet\n")},
	}
	if err := defaultPacketPublisher().publish(parent, packetName, oldPacket); err != nil {
		t.Fatal(err)
	}

	publisher := defaultPacketPublisher()
	realRename := publisher.rename
	publisher.rename = func(source, destination string) error {
		if strings.Contains(filepath.Base(source), ".stage-") && filepath.Base(destination) == packetName {
			return errors.New("injected publication failure")
		}
		return realRename(source, destination)
	}
	newPacket := applicationPacket{
		Manifest: packetManifest{SchemaVersion: packetSchemaVersion},
		Files:    map[string][]byte{"resume.md": []byte("new incomplete packet\n")},
	}
	if err := publisher.publish(parent, packetName, newPacket); err == nil || !strings.Contains(err.Error(), "injected publication failure") {
		t.Fatalf("publish() error = %v, want injected failure", err)
	}

	data, err := os.ReadFile(filepath.Join(parent, packetName, "resume.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "previous valid packet\n" {
		t.Fatalf("previous packet was not restored: %q", data)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".stage-") || strings.Contains(entry.Name(), ".backup-") {
			t.Fatalf("temporary packet directory remained after rollback: %s", entry.Name())
		}
	}
}

func TestPacketPublisherLeavesNoPacketOnInitialPublishFailure(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "applications")
	publisher := defaultPacketPublisher()
	publisher.rename = func(_, _ string) error { return errors.New("injected publication failure") }
	err := publisher.publish(parent, "example-engineer", applicationPacket{
		Manifest: packetManifest{SchemaVersion: packetSchemaVersion},
		Files:    map[string][]byte{"resume.md": []byte("not published\n")},
	})
	if err == nil {
		t.Fatal("publish() error = nil, want injected failure")
	}
	if _, err := os.Stat(filepath.Join(parent, "example-engineer")); !os.IsNotExist(err) {
		t.Fatalf("partial packet exists after failure: %v", err)
	}
}

func TestPacketPublisherCleansStagingAfterFileWriteFailure(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "applications")
	publisher := defaultPacketPublisher()
	realWrite := publisher.writeFile
	publisher.writeFile = func(path string, data []byte, mode os.FileMode) error {
		if filepath.Base(path) == "resume.md" {
			return errors.New("injected file write failure")
		}
		return realWrite(path, data, mode)
	}
	err := publisher.publish(parent, "example-engineer", applicationPacket{
		Manifest: packetManifest{SchemaVersion: packetSchemaVersion},
		Files: map[string][]byte{
			"cover_letter.md": []byte("staged first\n"),
			"resume.md":       []byte("write fails\n"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "injected file write failure") {
		t.Fatalf("publish() error = %v, want injected write failure", err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("staging content remained after write failure: %+v", entries)
	}
}

func TestPacketPublisherLeavesPreviousPacketOnStagingSyncFailure(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "applications")
	packetName := "example-engineer"
	previous := applicationPacket{
		Manifest: packetManifest{SchemaVersion: packetSchemaVersion},
		Files:    map[string][]byte{"resume.md": []byte("previous valid packet\n")},
	}
	if err := defaultPacketPublisher().publish(parent, packetName, previous); err != nil {
		t.Fatal(err)
	}

	publisher := defaultPacketPublisher()
	publisher.syncDir = func(path string) error {
		if strings.Contains(filepath.Base(path), ".stage-") {
			return errors.New("injected staging sync failure")
		}
		return syncDirectory(path)
	}
	err := publisher.publish(parent, packetName, applicationPacket{
		Manifest: packetManifest{SchemaVersion: packetSchemaVersion},
		Files:    map[string][]byte{"resume.md": []byte("replacement\n")},
	})
	if err == nil || !strings.Contains(err.Error(), "injected staging sync failure") {
		t.Fatalf("publish() error = %v, want injected sync failure", err)
	}
	data, err := os.ReadFile(filepath.Join(parent, packetName, "resume.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "previous valid packet\n" {
		t.Fatalf("previous packet changed after staging sync failure: %q", data)
	}
}
