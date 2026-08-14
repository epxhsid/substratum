package lsm

import (
	"encoding/binary"
	"io"
	"os"
	"sync"
)

type WAL struct {
	file *os.File
	mu   sync.RWMutex
}

func Open(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	w := &WAL{
		file: f,
	}
	return w, nil
}

func (w *WAL) Append(key, value string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	kb := []byte(key)
	vb := []byte(value)

	buf := make([]byte, 8+len(kb)+len(vb))
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(kb)))
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(vb)))
	copy(buf[8:], kb)
	copy(buf[8+len(kb):], vb)

	if _, err := w.file.Write(buf); err != nil {
		return err
	}

	return w.file.Sync()
}

func (w *WAL) Recover() (map[string]string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	data := make(map[string]string)

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	hbuf := make([]byte, 8)
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

		klen := binary.BigEndian.Uint32(hbuf[0:4])
		vlen := binary.BigEndian.Uint32(hbuf[4:8])

		kdata := make([]byte, klen)
		vdata := make([]byte, vlen)

		if _, err := io.ReadFull(w.file, kdata); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if err := w.file.Truncate(ofst); err != nil {
					return nil, err
				}
				break
			}
		}

		if _, err := io.ReadFull(w.file, vdata); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if err := w.file.Truncate(ofst); err != nil {
					return nil, err
				}
				break
			}
		}

		data[string(kdata)] = string(vdata)
	}

	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	return data, nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}
