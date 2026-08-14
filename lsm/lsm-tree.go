package lsm

import "sync"

type MemTable struct {
	data map[string]string
	size int
}

type SSTable struct {
	file string
}

type StorageEngine struct {
	mu              sync.RWMutex
	memTable        *MemTable
	sstf            []*SSTable
	wal             *WAL
	maxMemTableSize uint64
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

	return nil
}
