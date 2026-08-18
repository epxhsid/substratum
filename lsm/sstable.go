package lsm

import (
	"encoding/binary"
	"io"
	"os"
)

type SSTable struct {
	filePath string
	file     *os.File
	si       SparseIndex
}

func (s *SSTable) Get(key string) (*Entry, bool) {
	if s.file == nil {
		f, err := os.Open(s.filePath)
		if err != nil {
			return nil, false
		}
		s.file = f
	}

	file := s.file

	if err := s.si.load(file); err != nil {
		return nil, false
	}

	if _, err := file.Seek(s.si.offset(key), io.SeekStart); err != nil {
		return nil, false
	}

	hbuf := make([]byte, 9)

	for {
		if _, err := io.ReadFull(file, hbuf); err != nil {
			if err == io.EOF {
				return nil, false
			}
			return nil, false
		}

		op := hbuf[0]
		klen := binary.BigEndian.Uint32(hbuf[1:5])
		vlen := binary.BigEndian.Uint32(hbuf[5:9])

		kdata := make([]byte, klen)
		vdata := make([]byte, vlen)

		if _, err := io.ReadFull(file, kdata); err != nil {
			return nil, false
		}
		if _, err := io.ReadFull(file, vdata); err != nil {
			return nil, false
		}

		if string(kdata) != key {
			continue
		}

		switch op {
		case opSet:
			return &Entry{
				value:   string(vdata),
				deleted: false,
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
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
