package lsm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWAL_AppendAndRecover(t *testing.T) {
	dir := t.TempDir()
	wp := filepath.Join(dir, "wal.log")

	w1, err := Open(wp)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}

	if err := w1.Append("k1", "v1"); err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	w2, err := Open(wp)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}

	if err := w2.Append("k2", "v2"); err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	data, err := w2.Recover()
	if err != nil {
		t.Fatalf("failed to recover: %v", err)
	}
	w2.Close()
	if len(data) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(data))
	}

	if data["k1"] != "v1" {
		t.Fatalf("expected k1 to be v1, got %s", data["k1"])
	}
	if data["k2"] != "v2" {
		t.Fatalf("expected k2 to be v2, got %s", data["k2"])
	}
}

func TestWAL_OAppendFix(t *testing.T) {
	dir := t.TempDir()
	wp := filepath.Join(dir, "wal.log")

	w1, err := Open(wp)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}
	w1.Append("first_key", "first_val")
	w1.Close()

	w2, err := Open(wp)
	if err != nil {
		t.Fatalf("re-open failed: %v", err)
	}
	w2.Append("second_key", "second_val")

	data, err := w2.Recover()
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	w2.Close()

	if data["first_key"] != "first_val" || data["second_key"] != "second_val" {
		t.Fatalf("O_APPEND failed, data was overwritten: %+v", data)
	}
}

func TestWAL_TornWriteRecovery(t *testing.T) {
	dir := t.TempDir()
	wp := filepath.Join(dir, "wal.log")

	w, err := Open(wp)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}

	w.Append("valid1", "data1")
	w.Append("valid2", "data2")
	w.Close()

	f, err := os.OpenFile(wp, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("failed to open file for corrupt injection: %v", err)
	}
	f.Write([]byte{0x00, 0x00, 0x00, 0x04, 0x00})
	f.Close()

	w2, err := Open(wp)
	if err != nil {
		t.Fatalf("re-open failed: %v", err)
	}

	data, err := w2.Recover()
	if err != nil {
		t.Fatalf("recovery should handle torn write gracefully, got err: %v", err)
	}

	if len(data) != 2 {
		t.Fatalf("expected 2 valid items recovered, got %d", len(data))
	}

	if data["valid1"] != "data1" || data["valid2"] != "data2" {
		t.Errorf("recovered data corrupt: %+v", data)
	}

	if err := w2.Append("valid3", "data3"); err != nil {
		t.Fatalf("failed to append after torn write recovery: %v", err)
	}

	da, err := w2.Recover()
	if err != nil {
		t.Fatalf("post-append recovery failed: %v", err)
	}
	w2.Close()

	if len(da) != 3 {
		t.Fatalf("expected 3 items total, got %d", len(da))
	}
}
