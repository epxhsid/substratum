// SPDX-License-Identifier: MIT
package lsm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type SSTable struct {
	filePath string
	si       SparseIndex
	bf       *BloomFilter
}

func OpenSSTable(path string) (*SSTable, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	file.Close()

	bloomPath := path + ".bloom"
	data, err := os.ReadFile(bloomPath)
	if err != nil {
		return nil, err
	}

	bf := BloomFilter{}
	if err := bf.Unmarshal(data); err != nil {
		return nil, err
	}

	return &SSTable{
		filePath: path,
		bf:       &bf,
	}, nil
}

func (s *SSTable) Get(key string) (*Entry, bool, error) {
	if !s.bf.Contains(key) {
		return nil, false, nil
	}

	if err := s.si.load(s.filePath); err != nil {
		return nil, false, err
	}

	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	offset := s.si.offset(key)

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, false, err
	}

	for {
		rec, _, err := readRecord(file)

		if err == io.EOF {
			return nil, false, nil
		}

		if err != nil {
			return nil, false, err
		}

		if rec.key != key {
			if rec.key > key {
				return nil, false, nil
			}

			continue
		}

		switch rec.op {
		case opSet:
			return &Entry{
				value: rec.value,
			}, true, nil

		case opDelete:
			return &Entry{
				deleted: true,
			}, true, nil

		default:
			return nil, false, fmt.Errorf("lsm: invalid operation %d", rec.op)
		}
	}
}

func (s *SSTable) Close() error {
	return nil
}

func writeBloomFilter(path string, bf *BloomFilter) error {
	data, err := bf.Marshal()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".bloom-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}

	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}
