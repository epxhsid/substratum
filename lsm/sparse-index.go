package lsm

import (
	"encoding/binary"
	"io"
	"os"
	"sort"
	"sync"
)

type sparseIndexEntry struct {
	key    string
	offset int64
}

type SparseIndex struct {
	mu      sync.Mutex
	entries []sparseIndexEntry
	built   bool
	err     error
}

func (si *SparseIndex) load(file *os.File) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	if si.built {
		return si.err
	}

	entries, err := buildSparseIndex(file, sparseIndexInterval)
	if err != nil {
		si.err = err
		si.built = true
		return err
	}

	si.entries = entries.entries
	si.err = nil
	si.built = true
	return nil
}

func (si *SparseIndex) offset(key string) int64 {
	if len(si.entries) == 0 {
		return 0
	}

	idx := sort.Search(len(si.entries), func(i int) bool {
		return si.entries[i].key > key
	})

	if idx == 0 {
		return 0
	}

	return si.entries[idx-1].offset
}

func buildSparseIndex(file *os.File, interval int) (*SparseIndex, error) {
	if interval < 1 {
		interval = 1
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return &SparseIndex{}, err
	}

	index := SparseIndex{}
	header := make([]byte, 9)
	record := 0

	for {
		offset, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return &SparseIndex{}, err
		}

		if _, err := io.ReadFull(file, header); err != nil {
			if err == io.EOF {
				break
			}
			return &SparseIndex{}, err
		}

		klen := binary.BigEndian.Uint32(header[1:5])
		vlen := binary.BigEndian.Uint32(header[5:9])

		keyBuf := make([]byte, klen)
		valueBuf := make([]byte, vlen)

		if _, err := io.ReadFull(file, keyBuf); err != nil {
			return &SparseIndex{}, err
		}
		if _, err := io.ReadFull(file, valueBuf); err != nil {
			return &SparseIndex{}, err
		}

		if record%interval == 0 {
			index.entries = append(index.entries, sparseIndexEntry{
				key:    string(keyBuf),
				offset: offset,
			})
		}

		record++
	}

	return &index, nil
}
