// SPDX-License-Identifier: MIT
package lsm

import (
	"io"
	"os"
	"sort"
	"sync"
)

type sparseIndexEntry struct {
	key  string
	ofst int64
}

type SparseIndex struct {
	rw      sync.RWMutex
	entries []sparseIndexEntry
	built   bool
	err     error
}

func (si *SparseIndex) load(path string) error {
	si.rw.Lock()
	defer si.rw.Unlock()

	if si.built {
		return si.err
	}

	file, err := os.Open(path)
	if err != nil {
		si.err = err
		si.built = true
		return err
	}
	defer file.Close()

	entries, err := buildSparseIndex(file, sparseIndexInterval)
	if err != nil {
		si.err = err
		si.built = true
		return err
	}

	si.entries = entries
	si.err = nil
	si.built = true

	return nil
}

func (si *SparseIndex) offset(key string) int64 {
	si.rw.RLock()
	defer si.rw.RUnlock()

	if len(si.entries) == 0 {
		return 0
	}

	idx := sort.Search(
		len(si.entries),
		func(i int) bool {
			return si.entries[i].key > key
		},
	)

	if idx == 0 {
		return 0
	}

	return si.entries[idx-1].ofst
}

func buildSparseIndex(file *os.File, interval int) ([]sparseIndexEntry, error) {
	if interval < 1 {
		interval = 1
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var entries []sparseIndexEntry
	recordNumber := 0

	for {
		ofst, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		rec, _, err := readRecord(file)

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if recordNumber%interval == 0 {
			entries = append(entries, sparseIndexEntry{
				key:  rec.key,
				ofst: ofst,
			})
		}

		recordNumber++
	}
	return entries, nil
}
