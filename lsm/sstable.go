package lsm

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type SSTable struct {
	file string
}

func (s *SSTable) Get(key string) (*Entry, bool) {
	file, err := os.Open(s.file)
	if err != nil {
		return nil, false
	}
	defer file.Close()
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
		sstables = append(sstables, &SSTable{
			file: filepath.Join(dir, name),
		})
	}

	return sstables, nil
}
