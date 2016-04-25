package main

import (
	"github.com/fsnotify/fsnotify"

	"errors"
	"io"
	"os"
	"time"
)

type Reader struct {
	jsonFilename  string
	dataFilename  string
	writer        *Writer
	status        Status
	statusChanged chan Status
	file          *os.File
}

// NewReader reads cache entries from disk. If a writer is supplied, reads are
// synchronized to correspond with data becoming available.
func NewReader(writer *Writer, jsonFilename, dataFilename string) *Reader {
	r := &Reader{
		jsonFilename:  jsonFilename,
		dataFilename:  dataFilename,
		writer:        writer,
		status:        StatusNone,
		statusChanged: make(chan Status),
	}
	if r.writer {
		r.writer.Subscribe(r.statusChanged)
	}
	return r
}

// Retrieve the entry description from disk.
func (r *Reader) GetEntry() (*Entry, error) {
	if r.writer != nil {
		t := time.NewTimer(30 * time.Second)
		defer t.Stop()
	loop:
		for {
			select {
			case r.status = <-r.statusChanged:
				if r.status != StatusNone {
					break loop
				}
			case <-t.C:
				return nil, errors.New("timeout exceeded")
			}
		}
		if r.status == StatusError {
			return nil, errors.New("writer returned error")
		}
	}
	e := &Entry{}
	if err := e.LoadEntry(r.jsonFilename); err != nil {
		return nil, err
	}
	return e, nil
}

//
func (r *Reader) Open() (err error) {
	r.file, err = os.Open(r.dataFilename)
	return
}

// Read attempts to read from the file. For live reads, reads continue until
// the buffer is full or the writer status changes. fsnotify is used to keep
// track of new data being written to the file.
func (r *Reader) Read(p []byte) (n int, err error) {
	switch r.status {
	case StatusError:
		err = errors.New("writer error")
		return
	case StatusDone:
		err = os.EOF
		return
	default:
		for n < len(p) {
			var bytesRead int
			bytesRead, err = r.file.Read(p[n:])
			n += bytesRead
			if err != nil {
				if err == os.EOF && r.writer != nil {
					err = nil
					var watcher *fsnotify.Watcher
					watcher, err = fsnotify.NewWatcher()
					if err != nil {
						return
					}
					defer watcher.Close()
					if err = watcher.Add(r.dataFilename); err != nil {
						return
					}
				loop:
					for {
						select {
						case r.status = <-r.statusChanged:
							return
						case event := <-watcher.Events:
							if event.Op&fsnotify.Write == fsnotify.Write {
								break loop
							}
						}
					}
					continue
				}
				return
			}
		}
		return
	}
}

// Close cleans up any open resources.
func (r *Reader) Close() {
	if r.file != nil {
		r.file.Close()
	}
	if r.writer != nil {
		r.writer.Unsubscribe(r.statusChanged)
	}
}
