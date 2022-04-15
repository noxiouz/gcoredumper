package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/noxiouz/gcoredumper/configuration/configurator"
	_ "github.com/noxiouz/gcoredumper/configuration/configurator/localfile"
	"github.com/noxiouz/gcoredumper/core"
	"github.com/noxiouz/gcoredumper/report"
)

var (
	initialPid     = flag.Int64("P", 0, "initial PID")
	nsPid          = flag.Int64("p", 0, "ns PID")
	initialTid     = flag.Int64("I", 0, "initial TID")
	nsTid          = flag.Int64("i", 0, "ns TID")
	executable     = flag.String("E", "", "%E core_pattern")
	signalNum      = flag.Int("s", 0, "signal num %s")
	dumpable       = flag.Int("d", 0, "dumpable")
	timestampInSec = flag.Int64("t", 0, "")
	config         = flag.String("cfg", "embed:null", "path to config")
)

func SetUpLogger(w io.Writer) {
	log.SetOutput(w)
	log.SetPrefix(fmt.Sprintf("%v: ", uuid.NewString()))
}

func main() {
	flag.Parse()
	reporter := report.New()
	// TODO(noxiouz): make configurable
	cfg, err := configurator.Open("embed", "")
	if err != nil {
		log.Fatalf("%v", err)
	}

	config, err := cfg.Get(context.Background())
	if err != nil {
		log.Fatalf("%v", err)
	}
	f, err := os.OpenFile(config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println(err)
		SetUpLogger(io.Discard)
	} else {
		SetUpLogger(f)
		defer f.Close()
	}
	sink := &report.LogBasedReporter{Logger: log.Default()}
	defer reporter.Report(sink)
	log.Println("Start dump")

	ctx := report.WithReport(context.Background(), reporter)
	// TODO: add required arguments check
	si := core.SystemInput{
		// Pathname of Executable
		Executable: *executable,
		// TID in initial namespace
		InitialTid: *initialTid,
		// TID in process namespace
		NsTid: *nsTid,
		// PR_GET_DUMPABLE
		PrGetDumpable: core.Dumpable(*dumpable),
		// PID in intial namespace
		InitialPid: *initialPid,
		// PID in process namespace
		NsPid: *nsPid,
		// Time of dump
		DumpTime: time.Unix(*timestampInSec, 0),
		// Signal
		Signal: syscall.Signal(*signalNum),
		// Input
		Stream: os.Stdin,
	}

	err = core.Run(ctx, si, config)
	if err != nil {
		log.Println(err)
		reporter.AddError("core.run.error", err)
	}
}
