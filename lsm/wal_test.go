package lsm

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWALRecoverSetAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal")

	wal, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := wal.Set("alpha", "one"); err != nil {
		t.Fatal(err)
	}
	if err := wal.Set("beta", "two"); err != nil {
		t.Fatal(err)
	}
	if err := wal.Delete("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := wal.Close(); err != nil {
		t.Fatal(err)
	}

	wal, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()

	data, err := wal.Recover()
	if err != nil {
		t.Fatal(err)
	}

	if entry := data["alpha"]; entry == nil || !entry.deleted {
		t.Fatalf("expected alpha to recover as deleted, got %#v", entry)
	}
	if entry := data["beta"]; entry == nil || entry.deleted || entry.value != "two" {
		t.Fatalf("expected beta to recover with value %q, got %#v", "two", entry)
	}
}

func TestWALRecoverTruncatesPartialRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal")

	wal, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := wal.Set("alpha", "one"); err != nil {
		t.Fatal(err)
	}
	if err := wal.Close(); err != nil {
		t.Fatal(err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte{opSet, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	sizeBeforeRecover, err := fileSize(path)
	if err != nil {
		t.Fatal(err)
	}

	wal, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := wal.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if err := wal.Close(); err != nil {
		t.Fatal(err)
	}

	sizeAfterRecover, err := fileSize(path)
	if err != nil {
		t.Fatal(err)
	}

	if sizeAfterRecover >= sizeBeforeRecover {
		t.Fatalf("expected partial record to be truncated, before=%d after=%d", sizeBeforeRecover, sizeAfterRecover)
	}
	if entry := data["alpha"]; entry == nil || entry.deleted || entry.value != "one" {
		t.Fatalf("expected complete record to recover, got %#v", entry)
	}
}

func TestWALRecoverRejectsInvalidOperation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal")
	record := make([]byte, 9+len("alpha"))
	record[0] = 0xff
	binary.BigEndian.PutUint32(record[1:5], uint32(len("alpha")))
	copy(record[9:], "alpha")

	if err := os.WriteFile(path, record, 0644); err != nil {
		t.Fatal(err)
	}

	wal, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()

	if _, err := wal.Recover(); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("expected os.ErrInvalid, got %v", err)
	}
}

func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
