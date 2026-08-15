package lsm

import (
	"fmt"
	"os"
	"sync"
)

type StorageEngine struct {
	mu       sync.RWMutex
	memTable *MemTable
	sstf     []*SSTable
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

	memTable := &MemTable{
		data: data,
	}

	return &StorageEngine{
		wal:      wal,
		memTable: memTable,
		config:   config,
	}, nil
}

func (sc *StorageEngine) Set(key, value string) error {
	sc.memTable.data[key] = value
	if sc.wal != nil {
		if err := sc.wal.Append(key, value); err != nil {
			return err
		}
	}

	sc.memTable.Set(key, value)
	if sc.memTable.Size() > sc.config.MemTableSizeThreshold {
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
	sc.memTable = &MemTable{
		data: make(map[string]string),
	}

	file, err := os.CreateTemp(sc.config.DataDir, "lsm-sstable-*")
	if err != nil {
		return err
	}

	path := file.Name()

	for key, value := range old.data {
		if _, err := fmt.Fprintf(file, "%s\t%s\n", key, value); err != nil {
			file.Close()
			os.Remove(path)
			sc.memTable = old
		}
		return err
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

	sc.sstf = append(sc.sstf, &SSTable{
		file: path,
	})
	return nil
}
