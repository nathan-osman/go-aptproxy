package main

import (
	"github.com/hectane/go-asyncserver"
	"github.com/nathan-osman/go-aptproxy/cache"
	"github.com/pquerna/cachecontrol/cacheobject"

	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Server acts as an HTTP proxy, returning entries from the cache whenever
// possible. Recognized archive mirrors are silently rewritten to avoid
// needless duplication.
type Server struct {
	server *server.AsyncServer
	cache  *cache.Cache
}

func rewrite(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return rawurl
	}
	if strings.HasSuffix(u.Host, ".archive.ubuntu.com") {
		u.Host = "archive.ubuntu.com"
		rawurl = u.String()
	}
	return rawurl
}

func maxAge(req *http.Request) time.Duration {
	d, err := cacheobject.ParseRequestCacheControl(req.Header.Get("Cache-Control"))
	if err != nil {
		return time.Duration(d.MaxAge)
	}
	return 0
}

func (s *Server) writeHeaders(w http.ResponseWriter, e *cache.Entry) {
	l, err := strconv.ParseInt(e.ContentLength, 10, 64)
	if err == nil && l >= 0 {
		w.Header().Set("Content-Length", e.ContentLength)
	}
	w.Header().Set("Content-Type", e.ContentType)
	w.Header().Set("Last-Modified", e.LastModified)
	w.WriteHeader(http.StatusOK)
}

// ServeHTTP processes an incoming request to the proxy. GET requests are
// served with the storage backend and every other request is (out of
// necessity) rejected since it can't be cached.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r, err := s.cache.GetReader(rewrite(req.RequestURI), maxAge(req))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("[ERR]", err)
		return
	}
	defer r.Close()
	e, err := r.GetEntry()
	if err != nil {
		if dErr, ok := err.(*cache.DownloadError); ok {
			http.Error(w, dErr.Error(), http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		log.Println("[ERR]", err)
		return
	}
	s.writeHeaders(w, e)
	_, err = io.Copy(w, r)
	if err != nil {
		log.Println("[ERR]", err)
	}
}

// NewServer creates a new server.
func NewServer(addr, directory string) (*Server, error) {
	c, err := cache.NewCache(directory)
	if err != nil {
		return nil, err
	}
	s := &Server{
		server: server.New(addr),
		cache:  c,
	}
	s.server.Handler = s
	return s, nil
}

// Start initializes the server and begins listening for requests.
func (s *Server) Start() error {
	return s.server.Start()
}

// Stop shuts down the server.
func (s *Server) Stop() {
	s.server.Stop()
	s.cache.Close()
}
