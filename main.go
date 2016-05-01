package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		addr      = flag.String("addr", ":8000", "`host:port` to listen on")
		directory = flag.String("directory", "/var/cache/go-aptproxy", "`directory` used for storing packages")
	)
	flag.Parse()
	s, err := NewServer(*addr, *directory)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	s.Start()
	defer s.Stop()
	log.Println("APT proxy started")
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT)
	<-c
	log.Println("APT proxy shut down by signal")
}
