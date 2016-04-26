package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sync"
)

// Cache provides access to entries in the cache.
type Cache struct {
	mutex       sync.Mutex
	directory   string
	downloaders map[string]*Downloader
	waitGroup   sync.WaitGroup
}

// NewCache creates a new cache in the specified directory.
func NewCache(directory string) *Cache {
	return &Cache{
		directory:   directory,
		downloaders: make(map[string]*Downloader),
	}
}

// GetReader obtains an io.Reader for the specified rawurl. If a downloader
// currently exists for the URL, a live reader is created and connected to it.
// If the URL exists in the cache, it is read using the standard file API. If
// not, a downloader and live reader are created.
func (c *Cache) GetReader(rawurl string) (io.ReadCloser, chan *Entry, error) {
	var (
		b            = md5.Sum([]byte(rawurl))
		hash         = hex.EncodeToString(b[:])
		jsonFilename = path.Join(c.directory, fmt.Sprintf("%s.json", hash))
		dataFilename = path.Join(c.directory, fmt.Sprintf("%s.data", hash))
	)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	d, ok := c.downloaders[hash]
	if !ok {
		_, err := os.Stat(jsonFilename)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, nil, err
			}
		} else {
			e := &Entry{}
			if err = e.Load(jsonFilename); err != nil {
				return nil, nil, err
			}
			if e.Complete {
				f, err := os.Open(dataFilename)
				if err != nil {
					return nil, nil, err
				}
				eChan := make(chan *Entry)
				go func() {
					eChan <- e
					close(eChan)
				}()
				log.Println("[HIT]", rawurl)
				return f, eChan, nil
			}
		}
		d = NewDownloader(rawurl, jsonFilename, dataFilename)
		go func() {
			d.Wait()
			c.mutex.Lock()
			defer c.mutex.Unlock()
			delete(c.downloaders, hash)
			c.waitGroup.Done()
		}()
		c.downloaders[hash] = d
		c.waitGroup.Add(1)
	}
	eChan := make(chan *Entry)
	go func() {
		eChan <- d.GetEntry()
		close(eChan)
	}()
	log.Println("[MISS]", rawurl)
	return NewLiveReader(d, dataFilename), eChan, nil
}

// TODO: implement some form of "safe abort" for downloads so that the entire
// application doesn't end up spinning its tires waiting for downloads to end

// Close waits for all downloaders to complete before shutting down.
func (c *Cache) Close() {
	c.waitGroup.Wait()
}
