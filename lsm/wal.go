package lsm

import (
	"bufio"
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
	path     string
}

func Open(path string) (*WAL, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &WAL{
		file: f,
		buf:  bufio.NewWriter(f),
		path: path,
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

	buf, err := writeRecord(op, key, value)
	if err != nil {
		return err
	}

	n, err := w.buf.Write(buf)
	if err != nil {
		return err
	}
	w.bufBytes += n

	if w.bufBytes >= walFlushThreshold {
		return w.flushLocked()
	}

	return nil
}

func (w *WAL) Recover() (map[string]*Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.buf.Flush(); err != nil {
		return nil, err
	}

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	data := make(map[string]*Entry)

	for {
		ofst, err := w.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		rec, _, err := readRecord(w.file)
		if err == io.EOF {
			break
		}

		if err == io.ErrUnexpectedEOF {
			if err := w.file.Truncate(ofst); err != nil {
				return nil, err
			}
			break
		}

		if err != nil {
			return nil, err
		}

		switch rec.op {
		case opSet:
			data[rec.key] = &Entry{
				value: rec.value,
			}

		case opDelete:
			data[rec.key] = &Entry{
				value:   "",
				deleted: true,
			}
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

	return w.flushLocked()
}

func (w *WAL) flushLocked() error {
	if w.bufBytes == 0 {
		return nil
	}

	if err := w.buf.Flush(); err != nil {
		return err
	}

	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	w.bufBytes = 0
	return nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return nil
	}

	var firstErr error

	if err := w.buf.Flush(); err != nil {
		firstErr = err
	}

	if err := w.file.Sync(); err != nil && firstErr == nil {
		firstErr = err
	}

	if err := w.file.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	w.file = nil
	w.buf = nil

	return firstErr
}

func (w *WAL) Path() string {
	return w.path
}
