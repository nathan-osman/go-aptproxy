package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// Downloader attempts to download a file from a remote URL.
type Downloader struct {
	doneMutex  sync.Mutex
	err        error
	entry      *Entry
	entryMutex sync.Mutex
}

// NewDownloader creates a new downloader.
func NewDownloader(rawurl, jsonFilename, dataFilename string) *Downloader {
	d := &Downloader{}
	d.doneMutex.Lock()
	d.entryMutex.Lock()
	go func() {
		defer func() {
			d.doneMutex.Unlock()
		}()
		once := &sync.Once{}
		trigger := func() {
			once.Do(func() {
				d.entryMutex.Unlock()
			})
		}
		defer trigger()
		resp, err := http.Get(rawurl)
		if err != nil {
			d.err = err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			d.err = errors.New(resp.Status)
			return
		}
		f, err := os.Create(dataFilename)
		if err != nil {
			d.err = err
			return
		}
		defer f.Close()
		d.entry = &Entry{
			URL:           rawurl,
			ContentLength: strconv.FormatInt(resp.ContentLength, 10),
			ContentType:   resp.Header.Get("Content-Type"),
			LastModified:  resp.Header.Get("Last-Modified"),
		}
		if err = d.entry.Save(jsonFilename); err != nil {
			d.err = err
			return
		}
		trigger()
		n, err := io.Copy(f, resp.Body)
		if err != nil {
			d.err = err
			return
		}
		d.entry.ContentLength = strconv.FormatInt(n, 10)
		d.entry.Complete = true
		d.entry.Save(jsonFilename)
	}()
	return d
}

// GetEntry waits until the Entry associated with the download is available.
// This call will block until the entry is available or an error occurs.
func (d *Downloader) GetEntry() *Entry {
	d.entryMutex.Lock()
	defer d.entryMutex.Unlock()
	return d.entry
}

// Wait will block until the download completes.
func (d *Downloader) Wait() error {
	d.doneMutex.Lock()
	defer d.doneMutex.Unlock()
	return d.err
}
