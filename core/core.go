package core

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"

	"syscall"
	"time"

	"github.com/noxiouz/gcoredumper/bpfbacktracer"
	"github.com/noxiouz/gcoredumper/configuration"
	"github.com/noxiouz/gcoredumper/dumper"
	"github.com/noxiouz/gcoredumper/report"
	"github.com/spf13/afero"
)

type Dumpable int

const (
	DUMPABLE_DEFAULT  Dumpable = 0
	DUMPABLE_DEBUG    Dumpable = 1
	DUMPABLE_SUIDSAFE Dumpable = 2
)

func (d Dumpable) AllowCoreDump() bool {
	switch d {
	case DUMPABLE_DEBUG:
		return true
	case DUMPABLE_DEFAULT:
		return true
	case DUMPABLE_SUIDSAFE:
		return false
	default:
		return false
	}
}

type SystemInput struct {
	// Pathname of Executable
	Executable string
	// TID in initial namespace
	InitialTid int64
	// TID in process namespace
	NsTid int64
	// PR_GET_DUMPABLE
	PrGetDumpable Dumpable
	// PID in intial namespace
	InitialPid int64
	// PID in process namespace
	NsPid int64
	// Time of dump
	DumpTime time.Time
	// Signal
	Signal syscall.Signal
	// Input stream
	Stream io.ReadCloser
}

func skipCoredump() bool {
	return false
}

func Run(ctx context.Context, si SystemInput, config *configuration.Config) error {
	reporter := report.R(ctx)
	reporter.AddInt("signal", int64(si.Signal))

	pi, err := NewProcessInfo(ctx, si.InitialPid, si.NsPid, si.InitialTid, si.NsTid, afero.NewOsFs())
	if err != nil {
		return err
	}

	m, err := bpfbacktracer.LoadBacktracesMap()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer m.Close()
	var b bpfbacktracer.Backtrace
	m.Lookup(bpfbacktracer.Key(si.InitialTid), &b)
	for _, vaddr := range b.Vaddrs {
		log.Printf("Addr %x", vaddr)
	}

	if si.PrGetDumpable.AllowCoreDump() {
		if _, err := dumper.New(afero.NewOsFs()).Dump(ctx, si.Stream, filepath.Join(config.CorefilesDirectory, pi.CorefileName()), config.Dumper); err != nil {
			return err
		}
	} else {
		reporter.AddString("dump.status", "skipped")
	}
	return nil
}
