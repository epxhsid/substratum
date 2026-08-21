// SPDX-License-Identifier: MIT
package lsm

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type StorageEngine struct {
	mu       sync.RWMutex
	sstf     []*SSTable
	memTable *MemTable
	wal      *WAL
	config   *Config
}

func NewStorageEngine(config Config) (*StorageEngine, error) {
	if config.DataDir == "" {
		return nil, fmt.Errorf("data directory is empty")
	}

	if config.MemTableSizeThreshold <= 0 {
		return nil, fmt.Errorf("memtable threshold must be positive")
	}

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, err
	}

	walPath := filepath.Join(config.DataDir, "wal")

	wal, err := Open(walPath)
	if err != nil {
		return nil, err
	}

	data, err := wal.Recover()
	if err != nil {
		wal.Close()
		return nil, err
	}

	memTable := NewMemTable()

	for key, entry := range data {
		if entry.deleted {
			memTable.Delete(key)
		} else {
			memTable.Set(key, entry.value)
		}
	}

	sstables, err := loadSSTables(config.DataDir)
	if err != nil {
		wal.Close()
		return nil, err
	}

	return &StorageEngine{
		sstf:     sstables,
		memTable: memTable,
		wal:      wal,
		config:   &config,
	}, nil
}

func (sc *StorageEngine) Get(key string) (string, bool, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, ok := sc.memTable.Get(key)
	if ok {
		if entry.deleted {
			return "", false, nil
		}

		return entry.value, true, nil
	}

	for _, s := range slices.Backward(sc.sstf) {
		entry, ok, err := s.Get(key)
		if err != nil {
			return "", false, err
		}

		if !ok {
			continue
		}

		if entry.deleted {
			return "", false, nil
		}

		return entry.value, true, nil
	}

	return "", false, nil
}

func (sc *StorageEngine) Set(key, value string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if err := sc.wal.Set(key, value); err != nil {
		return err
	}

	sc.memTable.Set(key, value)

	if sc.memTable.size >= sc.config.MemTableSizeThreshold {
		return sc.flush()
	}

	return nil
}

func (sc *StorageEngine) Delete(key string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if err := sc.wal.Delete(key); err != nil {
		return err
	}

	sc.memTable.Delete(key)
	if sc.memTable.size >= sc.config.MemTableSizeThreshold {
		return sc.flush()
	}
	return nil
}

func (sc *StorageEngine) flush() error {
	if sc.memTable.Length() == 0 {
		return nil
	}

	old := sc.memTable

	if err := os.MkdirAll(sc.config.DataDir, 0755); err != nil {
		return err
	}

	keys := make([]string, 0, len(old.data))

	for key := range old.data {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	bf, err := NewBloomFilter(len(keys), 0.01)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(sc.config.DataDir, ".sstable-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}

	for _, key := range keys {
		entry := old.data[key]
		bf.Add(key)

		op := opSet
		value := entry.value

		if entry.deleted {
			op = opDelete
			value = ""
		}

		buf, err := writeRecord(op, key, value)
		if err != nil {
			cleanup()
			return err
		}

		if _, err := tmp.Write(buf); err != nil {
			cleanup()
			return err
		}
	}

	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	sstablePath, err := sc.nextSSTablePath()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, sstablePath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	bloomPath := sstablePath + ".bloom"

	if err := writeBloomFilter(bloomPath, bf); err != nil {
		os.Remove(sstablePath)
		return err
	}

	sstable, err := OpenSSTable(sstablePath)
	if err != nil {
		os.Remove(sstablePath)
		os.Remove(bloomPath)
		return err
	}

	if err := sc.wal.Reset(); err != nil {
		sstable.Close()
		os.Remove(sstablePath)
		os.Remove(bloomPath)
		return err
	}

	sc.sstf = append(sc.sstf, sstable)
	sc.memTable = NewMemTable()

	return nil
}

func (sc *StorageEngine) nextSSTablePath() (string, error) {
	entries, err := os.ReadDir(sc.config.DataDir)
	if err != nil {
		return "", err
	}

	maxID := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !strings.HasPrefix(name, "sstable-") {
			continue
		}

		idString := strings.TrimPrefix(name, "sstable-")

		id, err := strconv.Atoi(idString)
		if err != nil {
			continue
		}

		if id > maxID {
			maxID = id
		}
	}

	return filepath.Join(
		sc.config.DataDir,
		fmt.Sprintf("sstable-%020d", maxID+1),
	), nil
}

func loadSSTables(dir string) ([]*SSTable, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var names []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "sstable-") {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)

	sstables := make([]*SSTable, 0, len(names))

	for _, name := range names {
		path := filepath.Join(dir, name)

		sstable, err := OpenSSTable(path)
		if err != nil {
			return nil, err
		}

		sstables = append(sstables, sstable)
	}

	return sstables, nil
}

func (sc *StorageEngine) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.wal == nil {
		return nil
	}

	var firstErr error

	for _, s := range sc.sstf {
		if err := s.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if err := sc.wal.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	err := sc.wal.Close()
	sc.wal = nil

	return err
}
