//go:build windows

package main

import "os"

// Windows test-only fallback. Production is deployed on Linux and uses flock.
type runLock struct {
	file *os.File
	path string
}

func acquireLock(path string) (*runLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	return &runLock{file: f, path: path}, nil
}
func (l *runLock) Close() error { err := l.file.Close(); _ = os.Remove(l.path); return err }
