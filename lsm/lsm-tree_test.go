package lsm

import (
	"path/filepath"
	"testing"
)

func TestStorageEngineSetGetDelete(t *testing.T) {
	engine := newTestStorageEngine(t, 10)
	defer engine.wal.Close()

	if err := engine.Set("alpha", "one"); err != nil {
		t.Fatal(err)
	}

	value, ok := engine.Get("alpha")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if value != "one" {
		t.Fatalf("expected value %q, got %q", "one", value)
	}

	if err := engine.Delete("alpha"); err != nil {
		t.Fatal(err)
	}
	if value, ok := engine.Get("alpha"); ok {
		t.Fatalf("expected deleted key to be hidden, got %q", value)
	}
}

func TestStorageEngineFlushKeepsDataReadable(t *testing.T) {
	engine := newTestStorageEngine(t, 1)
	defer engine.wal.Close()

	if err := engine.Set("alpha", "one"); err != nil {
		t.Fatal(err)
	}

	if len(engine.sstf) != 1 {
		t.Fatalf("expected one sstable after flush, got %d", len(engine.sstf))
	}
	if engine.memTable.Size() != 0 {
		t.Fatalf("expected memtable to be empty after flush, got size %d", engine.memTable.Size())
	}

	value, ok := engine.Get("alpha")
	if !ok {
		t.Fatal("expected flushed key to be readable")
	}
	if value != "one" {
		t.Fatalf("expected value %q, got %q", "one", value)
	}
}

func TestStorageEngineRecoversFromWAL(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "wal")
	config := &Config{
		DataDir:               filepath.Join(dir, "data"),
		MemTableSizeThreshold: 10,
	}

	engine, err := NewStorageEngine(config, walPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Set("alpha", "one"); err != nil {
		t.Fatal(err)
	}
	if err := engine.wal.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, err := NewStorageEngine(config, walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.wal.Close()

	value, ok := recovered.Get("alpha")
	if !ok {
		t.Fatal("expected key to recover")
	}
	if value != "one" {
		t.Fatalf("expected value %q, got %q", "one", value)
	}
}

func newTestStorageEngine(t *testing.T, threshold int) *StorageEngine {
	t.Helper()

	dir := t.TempDir()
	engine, err := NewStorageEngine(&Config{
		DataDir:               filepath.Join(dir, "data"),
		MemTableSizeThreshold: threshold,
	}, filepath.Join(dir, "wal"))
	if err != nil {
		t.Fatal(err)
	}
	return engine
}
