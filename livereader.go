package main

import (
	"github.com/fsnotify/fsnotify"

	"io"
	"os"
)

// LiveReader synchronizes with a downloader to read from a file.
type LiveReader struct {
	dataFilename string
	open         chan bool
	done         chan error
	file         *os.File
	err          error
	eof          bool
}

// NewLiveReader creates a new live reader.
func NewLiveReader(d *Downloader, dataFilename string) *LiveReader {
	l := &LiveReader{
		dataFilename: dataFilename,
		open:         make(chan bool),
		done:         make(chan error),
	}
	go func() {
		d.GetEntry()
		close(l.open)
		l.done <- d.Wait()
		close(l.done)
	}()
	return l
}

// Read attempts to read data as it is being downloaded. If EOF is reached,
// fsnotify is used to watch for new data being written. The download is not
// complete until the "done" channel receives a value.
func (l *LiveReader) Read(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	<-l.open
	if l.file == nil {
		f, err := os.Open(l.dataFilename)
		if err != nil {
			return 0, err
		}
		l.file = f
	}
	var (
		bytesRead int
		watcher   *fsnotify.Watcher
	)
loop:
	for bytesRead < len(p) {
		n, err := l.file.Read(p[bytesRead:])
		bytesRead += n
		if err != nil {
			if err != io.EOF || l.eof {
				l.err = err
				break loop
			}
			if watcher == nil {
				watcher, err = fsnotify.NewWatcher()
				if err != nil {
					l.err = err
					break loop
				}
				defer watcher.Close()
				if err = watcher.Add(l.dataFilename); err != nil {
					l.err = err
					break loop
				}
				for {
					select {
					case e := <-watcher.Events:
						if e.Op&fsnotify.Write == fsnotify.Write {
							continue loop
						}
					case err = <-l.done:
						l.err = err
						l.eof = true
					}
				}
			}
		}
	}
	return bytesRead, l.err
}

// Close frees resources associated with the reader.
func (l *LiveReader) Close() error {
	l.file.Close()
	return nil
}
