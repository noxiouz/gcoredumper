package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"

	"github.com/noxiouz/gcoredumper/core"
	"github.com/noxiouz/gcoredumper/dumper"
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
	configPath     = flag.String("cfg", "./config.textproto", "path to config")
)

func skipCoredump() bool {
	return false
}

func main() {
	flag.Parse()
	// TODO(noxiouz): make configurable
	f, err := os.OpenFile("/var/log/gcoredumper.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.SetPrefix(fmt.Sprintf("%v: ", uuid.NewString()))
	reporter := report.New()
	sink := &report.LogBasedReporter{Logger: log.Default()}
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
	}

	config := core.LocalConfig{
		Dumper: &dumper.Configuraion{
			Compression:      dumper.Configuraion_PLANE,
			MaxDiskUsagePrct: 99,
		},
	}
	reporter.AddInt("signal", int64(*signalNum))

	pi, err := core.NewProcessInfo(ctx, si.InitialPid, si.NsPid, si.InitialTid, si.NsTid, afero.NewOsFs())
	if err != nil {
		log.Fatalf("faile to create ProcessInfo: %v", err)
	}
	// TODO(noxiouz): respect get dumpable and core limit.
	if !skipCoredump() {
		if _, err := dumper.New(afero.NewOsFs()).Dump(ctx, os.Stdin, filepath.Join("/tmp", pi.CorefileName()), config.Dumper); err != nil {
			log.Fatalf("failed to dump core %v", err)
		}
	}

	reporter.Report(sink)
}
