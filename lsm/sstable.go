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

func (s *SSTable) Get(key string) (*Entry, bool, error) {
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
			return nil, false, nil
		}
	}
}

func (s *SSTable) Close() error {
	return nil
}
