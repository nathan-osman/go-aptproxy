package main

import (
	"encoding/json"
	"os"
)

// Entry represents a file in storage.
type Entry struct {
	URL           string `json:"url"`
	ContentLength int64  `json:"content_length"`
	ContentType   string `json:"content_type"`
}

// LoadEntry loads an entry from disk.
func LoadEntry(filename string) (*Entry, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	e := &Entry{}
	if err = json.NewDecoder(f).Decode(e); err != nil {
		return nil, err
	}
	return e, nil
}

// Save writes the entry to disk.
func (e *Entry) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(e)
}
