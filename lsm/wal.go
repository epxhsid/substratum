package lsm

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sync"
)


type WAL struct {
	file     *os.File
	buf      *bufio.Writer
	mu       sync.RWMutex
	bufBytes int
}

func Open(path string) (*WAL, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &WAL{
		file: f,
		buf:  bufio.NewWriter(f),
	}, nil
}

func (w *WAL) Set(key, value string) error {
	return w.append(opSet, key, value)
}

func (w *WAL) Delete(key string) error {
	return w.append(opDelete, key, "")
}

func (w *WAL) append(op byte, key, value string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	kb := []byte(key)
	vb := []byte(value)

	buf := make([]byte, 9+len(kb)+len(vb))
	buf[0] = op
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(kb)))
	binary.BigEndian.PutUint32(buf[5:9], uint32(len(vb)))
	copy(buf[9:], kb)
	copy(buf[9+len(kb):], vb)

	n, err := w.buf.Write(buf)
	if err != nil {
		return err
	}
	w.bufBytes += n

	if w.bufBytes >= walFlushThreshold {
		if err := w.buf.Flush(); err != nil {
			return err
		}
		if err := w.file.Sync(); err != nil {
			return err
		}
		w.bufBytes = 0
	}

	return nil
}

func (w *WAL) Recover() (map[string]*Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buf != nil {
		if err := w.buf.Flush(); err != nil {
			return nil, err
		}
	}

	data := make(map[string]*Entry)

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	hbuf := make([]byte, 9)
	for {
		ofst, err := w.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		if _, err := io.ReadFull(w.file, hbuf); err != nil {
			if err == io.EOF {
				break
			}
			if err == io.ErrUnexpectedEOF {
				if err := w.file.Truncate(ofst); err != nil {
					return nil, err
				}
				break
			}
			return nil, err
		}

		op := hbuf[0]
		klen := binary.BigEndian.Uint32(hbuf[1:5])
		vlen := binary.BigEndian.Uint32(hbuf[5:9])

		kdata := make([]byte, klen)
		vdata := make([]byte, vlen)

		if _, err := io.ReadFull(w.file, kdata); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if err := w.file.Truncate(ofst); err != nil {
					return nil, err
				}
				break
			}
			return nil, err
		}

		if _, err := io.ReadFull(w.file, vdata); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if err := w.file.Truncate(ofst); err != nil {
					return nil, err
				}
				break
			}
			return nil, err
		}

		key := string(kdata)

		switch op {
		case opSet:
			data[key] = &Entry{
				value:   string(vdata),
				deleted: false,
			}
		case opDelete:
			data[key] = &Entry{
				value:   "",
				deleted: true,
			}

		default:
			return nil, os.ErrInvalid
		}
	}

	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	return data, nil
}

func (w *WAL) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf != nil {
		if err := w.buf.Flush(); err != nil {
			return err
		}
		if err := w.file.Sync(); err != nil {
			return err
		}
		w.bufBytes = 0
	}
	return nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf != nil {
		if err := w.buf.Flush(); err != nil {
			return err
		}
	}
	return w.file.Close()
}
