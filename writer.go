package main

import (
	"container/list"
	"io"
	"net/http"
	"os"
	"sync"
)

// Status of download in progress.
type Status int

const (
	StatusNone        = iota // Nothing has happened yet
	StatusDownloading        // JSON and data files created
	StatusError              // Error retrieving the data
	StatusDone               // All data has been retrieved
)

// Writer connects to a remote URL via HTTP and retrieves the data, writing it
// directly to disk and notifying subscribed readers of its status.
type Writer struct {
	mutex    sync.Mutex
	channels *list.List
	status   Status
}

func (w *Writer) sendStatus(statusChan chan<- Status, status Status) {
	statusChan <- status
}

func (w *Writer) setStatus(status Status) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.status = status
	for e := w.channels.Front(); e != nil; e = e.Next() {
		go w.sendStatus(e.Value.(chan<- Status), status)
	}
}

// NewWriter creates a new writer for the given URL. Status information is
// passed along to subscribed channels. The done channel is used to notify the
// storage system that the download is complete (either success or an error).
func NewWriter(url, jsonFilename, dataFilename string, done chan<- *Writer) *Writer {
	w := &Writer{
		channels: list.New(),
	}
	go func() {
		r, err := http.Get(url)
		if err != nil {
			goto error
		}
		defer r.Body.Close()
		e := &Entry{
			URL:           url,
			ContentLength: r.ContentLength,
			ContentType:   r.Header.Get("Content-Type"),
		}
		if err = e.Save(jsonFilename); err != nil {
			goto error
		}
		f, err := os.Create(dataFilename)
		if err != nil {
			goto error
		}
		defer f.Close()
		w.setStatus(StatusDownloading)
		_, err = io.Copy(f, r.Body)
		if err != nil {
			goto error
		}
		w.setStatus(StatusDone)
		done <- w
	error:
		w.setStatus(StatusError)
		done <- w
	}()
	return w
}

// Subscribe adds a channel to the list to be notified when the writer's status
// changes. The channel will also immediately receive the current status.
func (w *Writer) Subscribe(statusChan chan<- Status) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.channels.PushBack(statusChan)

	// TODO: is "go" necessary here?
	go w.sendStatus(statusChan, w.status)
}

// Unsubscribe removes a channel from the list to be notified. This may occur
// when a client cancels a request, for example.
func (w *Writer) Unsubscribe(statusChan chan<- Status) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for e := w.channels.Front(); e != nil; e = e.Next() {
		if e.Value.(chan<- Status) == statusChan {
			w.channels.Remove(e)
		}
	}
}
