package cache

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// DownloadError conveys information about a download request that failed.
type DownloadError struct {
	Status string
}

// Error returns a description of the error.
func (d *DownloadError) Error() string {
	return d.Status
}

// downloader attempts to download a file from a remote URL.
type downloader struct {
	doneMutex  sync.Mutex
	err        error
	entry      *Entry
	entryMutex sync.Mutex
}

// newDownloader creates a new downloader.
func newDownloader(rawurl, jsonFilename, dataFilename string) *downloader {
	d := &downloader{}
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
			d.err = &DownloadError{
				Status: resp.Status,
			}
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

// GetEntry retrieves the entry associated with the download.
func (d *downloader) GetEntry() (*Entry, error) {
	d.entryMutex.Lock()
	defer d.entryMutex.Unlock()
	return d.entry, d.err
}

// WaitForDone will block until the download completes.
func (d *downloader) WaitForDone() error {
	d.doneMutex.Lock()
	defer d.doneMutex.Unlock()
	return d.err
}
