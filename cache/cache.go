package cache

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

// Reader is a generic interface for reading cache entries either from disk or
// directly attached to a downloader.
type Reader interface {
	io.ReadCloser
	GetEntry() (*Entry, error)
}

// Cache provides access to entries in the cache.
type Cache struct {
	mutex       sync.Mutex
	directory   string
	downloaders map[string]*downloader
	waitGroup   sync.WaitGroup
}

// NewCache creates a new cache in the specified directory.
func NewCache(directory string) (*Cache, error) {
	if err := os.MkdirAll(directory, 0775); err != nil {
		return nil, err
	}
	return &Cache{
		directory:   directory,
		downloaders: make(map[string]*downloader),
	}, nil
}

// GetReader obtains a Reader for the specified rawurl. If a downloader
// currently exists for the URL, a live reader is created and connected to it.
// If the URL exists in the cache, it is read using the standard file API. If
// not, a downloader and live reader are created.
func (c *Cache) GetReader(rawurl string) (Reader, error) {
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
				return nil, err
			}
		} else {
			r, err := newDiskReader(jsonFilename, dataFilename)
			if err != nil {
				return nil, err
			}
			if e, _ := r.GetEntry(); e.Complete {
				log.Println("[HIT]", rawurl)
				return r, nil
			}
		}
		d = newDownloader(rawurl, jsonFilename, dataFilename)
		go func() {
			d.WaitForDone()
			c.mutex.Lock()
			defer c.mutex.Unlock()
			delete(c.downloaders, hash)
			c.waitGroup.Done()
		}()
		c.downloaders[hash] = d
		c.waitGroup.Add(1)
	}
	log.Println("[MISS]", rawurl)
	return newLiveReader(d, dataFilename)
}

// TODO: implement some form of "safe abort" for downloads so that the entire
// application doesn't end up spinning its tires waiting for downloads to end

// Close waits for all downloaders to complete before shutting down.
func (c *Cache) Close() {
	c.waitGroup.Wait()
}
