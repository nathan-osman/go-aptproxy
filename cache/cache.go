package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
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

// getFilenames returns the filenames for the JSON and data files from a URL.
func (c *Cache) getFilenames(rawurl string) (hash, jsonFilename, dataFilename string) {
	b := md5.Sum([]byte(rawurl))
	hash = hex.EncodeToString(b[:])
	jsonFilename = path.Join(c.directory, fmt.Sprintf("%s.json", hash))
	dataFilename = path.Join(c.directory, fmt.Sprintf("%s.data", hash))
	return
}

// GetReader obtains a Reader for the specified rawurl. If a downloader
// currently exists for the URL, a live reader is created and connected to it.
// If the URL exists in the cache, it is read using the standard file API. If
// not, a downloader and live reader are created.
func (c *Cache) GetReader(rawurl string, maxAge time.Duration) (Reader, error) {
	hash, jsonFilename, dataFilename := c.getFilenames(rawurl)
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
			e, _ := r.GetEntry()
			lastModified, _ := time.Parse(http.TimeFormat, e.LastModified)
			if lastModified.Before(time.Now().Add(maxAge)) || e.Complete {
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

// Insert adds an item into the cache.
func (c *Cache) Insert(rawurl string, r io.Reader) error {
	_, jsonFilename, dataFilename := c.getFilenames(rawurl)
	f, err := os.Open(dataFilename)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	e := &Entry{
		URL:           rawurl,
		Complete:      true,
		ContentLength: strconv.FormatInt(n, 10),
		ContentType:   mime.TypeByExtension(rawurl),
		LastModified:  time.Now().Format(http.TimeFormat),
	}
	return e.Save(jsonFilename)
}

// TODO: implement some form of "safe abort" for downloads so that the entire
// application doesn't end up spinning its tires waiting for downloads to end.

// Close waits for all downloaders to complete before shutting down.
func (c *Cache) Close() {
	c.waitGroup.Wait()
}
