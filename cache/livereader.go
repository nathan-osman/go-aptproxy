package cache

import (
	"github.com/fsnotify/fsnotify"

	"io"
	"os"
)

// liveReader reads a file from disk, synchronizing reads with a downloader.
type liveReader struct {
	downloader   *downloader
	dataFilename string
	file         *os.File
	entry        *Entry
	done         chan error
	err          error
	eof          bool
}

// newLiveReader creates a reader from the provided downloader and data
// file. fsnotify is used to watch for writes to the file to avoid using a
// spinloop. Invoking this function assumes the existence of the data file.
func newLiveReader(d *downloader, dataFilename string) (*liveReader, error) {
	l := &liveReader{
		downloader:   d,
		dataFilename: dataFilename,
		done:         make(chan error),
	}
	go func() {
		defer close(l.done)
		l.done <- d.WaitForDone()
	}()
	return l, nil
}

// Read attempts to read as much data as possible into the provided buffer.
// Since data is being downloaded as data is being read, fsnotify is used to
// monitor writes to the file. This function blocks until the requested amount
// of data is read, an error occurs, or EOF is encountered.
func (l *liveReader) Read(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	if l.file == nil {
		f, err := os.Open(l.dataFilename)
		if err != nil {
			l.err = err
			return 0, l.err
		}
		l.file = f
	}
	bytesRead := 0
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		l.err = err
		return 0, l.err
	}
	defer watcher.Close()
	if err := watcher.Add(l.dataFilename); err != nil {
		l.err = err
		return 0, l.err
	}
loop:
	for bytesRead < len(p) {
		n, err := l.file.Read(p[bytesRead:])
		bytesRead += n
		if err != nil {
			if err != io.EOF || l.eof {
				l.err = err
				break loop
			}
			for {
				select {
				case e := <-watcher.Events:
					if e.Op&fsnotify.Write != fsnotify.Write {
						continue
					}
				case err = <-l.done:
					l.err = err
					l.eof = true
				}
				continue loop
			}
		}
	}
	return bytesRead, l.err
}

// Close attempts to close the data file (if opened).
func (l *liveReader) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetEntry returns the Entry associated with the file, blocking until either
// the data is available or an error occurs.
func (l *liveReader) GetEntry() (*Entry, error) {
	return l.downloader.GetEntry()
}
