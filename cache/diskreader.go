package cache

import (
	"os"
)

// diskReader reads a file from the cache on disk.
type diskReader struct {
	entry *Entry
	file  *os.File
}

// newDiskReader creates a reader from the provided JSON and data filenames.
// Failure to open either of these results in an immediate error.
func newDiskReader(jsonFilename, dataFilename string) (*diskReader, error) {
	e := &Entry{}
	if err := e.Load(jsonFilename); err != nil {
		return nil, err
	}
	f, err := os.Open(dataFilename)
	if err != nil {
		return nil, err
	}
	return &diskReader{
		entry: e,
		file:  f,
	}, nil
}

// Read attempts to read as much data as possible into the provided buffer.
func (d *diskReader) Read(p []byte) (int, error) {
	return d.file.Read(p)
}

// Close attempts to close the data file.
func (d *diskReader) Close() error {
	return d.file.Close()
}

// GetEntry returns the Entry associated with the file.
func (d *diskReader) GetEntry() (*Entry, error) {
	return d.entry, nil
}
