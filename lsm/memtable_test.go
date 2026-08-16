package lsm

import "testing"

func TestMemTableSetGetAndDelete(t *testing.T) {
	memTable := &MemTable{
		data: make(map[string]*Entry),
	}

	memTable.Set("alpha", "one")

	value, ok := memTable.Get("alpha")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if value != "one" {
		t.Fatalf("expected value %q, got %q", "one", value)
	}

	memTable.Delete("alpha")

	if value, ok := memTable.Get("alpha"); ok {
		t.Fatalf("expected deleted key to be hidden, got %q", value)
	}
	if memTable.Size() != 1 {
		t.Fatalf("expected tombstone to remain in memtable, got size %d", memTable.Size())
	}
}
