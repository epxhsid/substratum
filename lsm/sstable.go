package lsm

import (
	"io"
	"os"
)

type SSTable struct {
	filePath string
	si       SparseIndex
}

func OpenSSTable(path string) (*SSTable, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	file.Close()

	return &SSTable{
		filePath: path,
	}, nil
}

func (s *SSTable) Get(key string) (*Entry, bool) {
	if err := s.si.load(s.filePath); err != nil {
		return nil, false
	}

	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	offset := s.si.offset(key)

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, false
	}

	for {
		rec, _, err := readRecord(file)

		if err == io.EOF {
			return nil, false
		}

		if err != nil {
			return nil, false
		}

		if rec.key != key {
			if rec.key > key {
				return nil, false
			}

			continue
		}

		switch rec.op {
		case opSet:
			return &Entry{
				value: rec.value,
			}, true

		case opDelete:
			return &Entry{
				deleted: true,
			}, true

		default:
			return nil, false
		}
	}
}

func (s *SSTable) Close() error {
	return nil
}
