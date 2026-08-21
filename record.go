// SPDX-License-Identifier: MIT
package lsm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	ErrCorruptRecord = errors.New("lsm: corrupt record")
	ErrInvalidRecord = errors.New("lsm: invalid record")
)

type record struct {
	op    byte
	key   string
	value string
}

func writeRecord(op byte, key, value string) ([]byte, error) {
	if len(key) > maxKeySize {
		return nil, fmt.Errorf("key too large: %d bytes", len(key))
	}

	if len(value) > maxValueSize {
		return nil, fmt.Errorf("value too large: %d bytes", len(value))
	}

	if op != opSet && op != opDelete {
		return nil, ErrInvalidRecord
	}

	if op == opDelete {
		value = ""
	}

	buf := make([]byte, 9+len(key)+len(value))

	buf[0] = op

	binary.BigEndian.PutUint32(buf[1:5], uint32(len(key)))
	binary.BigEndian.PutUint32(buf[5:9], uint32(len(value)))
	copy(buf[9:], key)
	copy(buf[9+len(key):], value)

	return buf, nil
}

func readRecord(r io.Reader) (record, int, error) {
	header := make([]byte, 9)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return record{}, 0, err
	}

	op := header[0]
	klen := binary.BigEndian.Uint32(header[1:5])
	vlen := binary.BigEndian.Uint32(header[5:9])

	if op != opSet && op != opDelete {
		return record{}, 0, ErrCorruptRecord
	}

	if klen > maxKeySize || vlen > maxValueSize {
		return record{}, 0, ErrCorruptRecord
	}

	total := 9 + int(klen) + int(vlen)

	key := make([]byte, klen)
	if _, err := io.ReadFull(r, key); err != nil {
		return record{}, 0, err
	}

	value := make([]byte, vlen)
	if _, err := io.ReadFull(r, value); err != nil {
		return record{}, 0, err
	}

	if op == opDelete && len(value) != 0 {
		return record{}, 0, ErrCorruptRecord
	}

	return record{
		op:    op,
		key:   string(key),
		value: string(value),
	}, total, nil
}
