package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	s, err := NewServer(":8000", "/tmp/proxy")
	if err != nil {
		panic(err)
	}
	s.Start()
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT)
	<-c
	s.Stop()
}
