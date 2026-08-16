package lsm

import (
	"encoding/binary"
	"io"
	"os"
)

type SSTable struct {
	file string
}

func (s *SSTable) Get(key string) (string, bool) {
	file, err := os.Open(s.file)
	if err != nil {
		return "", false
	}
	defer file.Close()

	hbuf := make([]byte, 9)
	for {
		if _, err := io.ReadFull(file, hbuf); err != nil {
			return "", false
		}

		op := hbuf[0]
		klen := binary.BigEndian.Uint32(hbuf[1:5])
		vlen := binary.BigEndian.Uint32(hbuf[5:9])

		kdata := make([]byte, klen)
		vdata := make([]byte, vlen)

		if _, err := io.ReadFull(file, kdata); err != nil {
			return "", false
		}
		if _, err := io.ReadFull(file, vdata); err != nil {
			return "", false
		}

		if string(kdata) != key {
			continue
		}

		if op == opDelete {
			return "", false
		}
		return string(vdata), true
	}
}
