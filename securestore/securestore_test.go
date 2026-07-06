package securestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetGetDelete(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "emo-securestore-test")
	defer os.RemoveAll(tmpDir)

	if err := Init(tmpDir); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Set
	if err := Set("token", "abc123"); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Get
	val, ok := Get("token")
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if val != "abc123" {
		t.Fatalf("Get = %q, want abc123", val)
	}

	// Has
	if !Has("token") {
		t.Fatal("Has should return true")
	}

	// Delete
	if err := Delete("token"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if Has("token") {
		t.Fatal("Has should return false after delete")
	}
}

func TestPersistence(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "emo-securestore-test")
	defer os.RemoveAll(tmpDir)

	// First init + set
	Init(tmpDir)
	Set("key1", "value1")

	// Verify file exists
	storePath := filepath.Join(tmpDir, ".emo", "securestore.json")
	if _, err := os.Stat(storePath); err != nil {
		t.Fatalf("store file not created: %v", err)
	}

	// Re-init (simulating app restart) and verify value persists
	// Note: once.Do prevents re-init, so we directly load
	instance.load()
	val, ok := Get("key1")
	if !ok {
		t.Fatal("value not persisted")
	}
	if val != "value1" {
		t.Fatalf("persisted value = %q, want value1", val)
	}
}

func TestKeys(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "emo-securestore-test")
	defer os.RemoveAll(tmpDir)

	Init(tmpDir)
	Set("a", "1")
	Set("b", "2")
	Set("c", "3")

	keys := Keys()
	if len(keys) != 3 {
		t.Fatalf("len(keys) = %d, want 3", len(keys))
	}
}

func TestClear(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "emo-securestore-test")
	defer os.RemoveAll(tmpDir)

	Init(tmpDir)
	Set("a", "1")
	Set("b", "2")

	Clear()

	if len(Keys()) != 0 {
		t.Fatal("Keys should be empty after Clear")
	}
}
