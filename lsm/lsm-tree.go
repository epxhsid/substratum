package lsm

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"sort"
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

func NewStorageEngine(config *Config, path string) (*StorageEngine, error) {
	wal, err := Open(path)
	if err != nil {
		return nil, err
	}

	data, err := wal.Recover()
	if err != nil {
		wal.Close()
		return nil, err
	}

	if data == nil {
		data = make(map[string]*Entry)
	}

	sstables, err := loadSSTables(config.DataDir)
	if err != nil {
		wal.Close()
		return nil, err
	}

	memTable := &MemTable{
		data: data,
	}

	return &StorageEngine{
		wal:      wal,
		memTable: memTable,
		sstf:     sstables,
		config:   config,
	}, nil
}

func (sc *StorageEngine) Get(key string) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if value, ok := sc.memTable.Get(key); ok {
		return value, true
	}

	for _, s := range slices.Backward(sc.sstf) {
		entry, ok := s.Get(key)
		if !ok {
			continue
		}

		if entry.deleted {
			return "", false
		}

		return entry.value, true
	}

	return "", false
}

func (sc *StorageEngine) Set(key, value string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if err := sc.wal.Set(key, value); err != nil {
		return err
	}

	sc.memTable.Set(key, value)
	if sc.memTable.Size() >= sc.config.MemTableSizeThreshold {
		if err := sc.flush(); err != nil {
			return err
		}
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
	if sc.memTable.Size() >= sc.config.MemTableSizeThreshold {
		if err := sc.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (sc *StorageEngine) flush() error {
	if err := os.MkdirAll(sc.config.DataDir, 0755); err != nil {
		return err
	}

	old := sc.memTable
	file, err := os.CreateTemp(sc.config.DataDir, "lsm-sstable-*")
	if err != nil {
		return err
	}

	path := file.Name()

	keys := make([]string, 0, len(old.data))
	for key := range old.data {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		entry := old.data[key]

		buf := make([]byte, 9+len(key)+len(entry.value))

		if entry.deleted {
			buf[0] = opDelete
		} else {
			buf[0] = opSet
		}

		binary.BigEndian.PutUint32(buf[1:5], uint32(len(key)))
		binary.BigEndian.PutUint32(buf[5:9], uint32(len(entry.value)))
		copy(buf[9:], key)
		copy(buf[9+len(key):], entry.value)

		if _, err := file.Write(buf); err != nil {
			file.Close()
			os.Remove(path)
			sc.memTable = old
			return err
		}
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(path)
		sc.memTable = old
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(path)
		sc.memTable = old
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}

	sstable := &SSTable{
		filePath: path,
		file:     f,
	}
	sc.sstf = append(sc.sstf, sstable)

	sc.memTable = &MemTable{
		data: make(map[string]*Entry),
	}

	return nil
}

func loadSSTables(dir string) ([]*SSTable, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "lsm-sstable-") {
			files = append(files, entry.Name())
		}
	}

	slices.Sort(files)

	sstables := make([]*SSTable, 0, len(files))

	for _, name := range files {
		path := filepath.Join(dir, name)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sstables = append(sstables, &SSTable{
			filePath: path,
			file:     f,
		})
	}

	return sstables, nil
}

func (sc *StorageEngine) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for _, s := range sc.sstf {
		if s != nil && s.file != nil {
			s.file.Close()
			s.file = nil
		}
	}
	if sc.wal != nil {
		sc.wal.Flush()
		return sc.wal.Close()
	}
	return nil
}
