package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"sync"
)

// Storage provides read and write access to items in the cache. In order to
// avoid race conditions, adding and testing for entries in the cache are
// protected by a mutex.
type Storage struct {
	directory  string
	writers    map[string]*Writer
	writerDone chan *Writer
	mutex      sync.Mutex
}

// NewStorage creates a new storage manager.
func NewStorage(directory string) *Storage {
	return &Storage{
		directory:  directory,
		writers:    make(map[string]*Writer),
		writerDone: make(chan *Writer),
	}
}

// GetReader returns a *Reader for the specified URL. If the file does not
// exist, both a writer (for downloading the file) and a reader are created.
func (s *Storage) GetReader(url string) (*Reader, error) {
	var (
		hash         = string(md5.Sum([]byte(url)))
		jsonFilename = path.Join(s.directory, fmt.Sprintf("%s.json", hash))
		dataFilename = path.Join(s.directory, fmt.Sprintf("%s.data", hash))
	)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	w, ok := s.writers[hash]
	if ok {
		return NewReader(w, jsonFilename, dataFilename), nil
	} else {
		_, err := os.Stat(jsonFilename)
		if err != nil {
			if os.IsNotExist(err) {
				w = NewWriter(url, jsonFilename, dataFilename, s.writerDone)
				s.writers[hash] = w
				return NewReader(w, jsonFilename, dataFilename), nil
			} else {
				return nil, err
			}
		} else {
			return NewReader(nil, jsonFilename, dataFilename), nil
		}
	}
}
