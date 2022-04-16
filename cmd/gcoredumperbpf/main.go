package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/noxiouz/gcoredumper/bpfbacktracer"
)

func main() {
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	if err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	}); err != nil {
		log.Fatalf("setting temporary rlimit: %s", err)
	}
	b, err := bpfbacktracer.NewBPFBacktracer()
	if err != nil {
		log.Fatalf("backtracker.New failed: %v", err)
	}
	defer b.Close()
	log.Println("Stand by... Keeping KProbe alive")
	<-stopper
}
