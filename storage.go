package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
)

// Storage provides read and write access to items in the cache. In order to
// avoid race conditions, adding and testing for entries in the cache are
// protected by a mutex.
type Storage struct {
	Directory string
	mutex     sync.Mutex
}

// GetReader returns an io.Reader for the specified URL. If the file does not
// exist, both a writer (for downloading the file) and a reader are created.
func (s *Storage) GetReader(url string) (io.Reader, error) {
	var (
		hash         = string(md5.Sum([]byte(url)))
		jsonFilename = path.Join(s.Directory, fmt.Sprintf("%s.json", hash))
		dataFilename = path.Join(s.Directory, fmt.Sprintf("%s.data", hash))
	)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	i, err := os.Stat(jsonFilename)
	if err != nil {
		if os.IsNotExist(err) {
			NewWriter(url, jsonFilename, dataFilename)
		} else {
			return nil, err
		}
	}
	return NewReader(url, jsonFilename, dataFilename)
}
