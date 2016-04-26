package main

import (
	"encoding/json"
	"os"
)

// Entry represents an individual item in the cache.
type Entry struct {
	URL           string `json:"url"`
	Complete      bool   `json:"complete"`
	ContentLength string `json:"content_length"`
	ContentType   string `json:"content_type"`
	LastModified  string `json:"last_modified"`
}

// Load reads the entry from disk.
func (e *Entry) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(e)
}

// Save writes the entry to disk.
func (e *Entry) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(e)
}
