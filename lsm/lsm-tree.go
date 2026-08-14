package lsm

import "sync"

type StorageEngine struct {
	mu              sync.RWMutex
	memTable        *MemTable
	sstf            []*SSTable
	wal             *WAL
	config          *Config
}

func NewStorageEngine(path string) (*StorageEngine, error) {
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
	if sc.memTable.Si

	return nil
}
