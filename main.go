package main

import (
	"github.com/micro/mdns"

	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		host      = flag.String("host", "0.0.0.0", "`host` to listen on")
		port      = flag.Int("port", 8000, "`port` to listen on")
		directory = flag.String("directory", "/var/cache/go-aptproxy", "`directory` used for storing packages")
	)
	flag.Parse()

	// Create the HTTP server and initialize the cache
	addr := fmt.Sprintf("%s:%d", *host, *port)
	httpServer, err := NewServer(addr, *directory)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println("Starting HTTP server...")
	if err = httpServer.Start(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer func() {
		httpServer.Stop()
		log.Println("Stopping HTTP server...")
	}()

	// Respond to mDNS queries
	h, err := os.Hostname()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	service, err := mdns.NewMDNSService(
		h,
		"_apt_proxy._tcp",
		"",
		"",
		*port,
		nil,
		[]string{"go-aptproxy - Smarter APT Proxy"},
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println("Starting mDNS server...")
	mDNSServer, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer func() {
		mDNSServer.Shutdown()
		log.Println("Stopping mDNS server...")
	}()

	// Wait for a signal
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT)
	<-c
	log.Println("Caught SIGINT")
}
