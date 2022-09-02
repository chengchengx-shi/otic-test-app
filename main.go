package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println(os.Getpid())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	<-ch
}
