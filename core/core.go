package core

import (
	"context"
	"io"
	"path/filepath"

	"syscall"
	"time"

	dumper "github.com/noxiouz/gcoredumper/dumper"
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

func Run(ctx context.Context, si SystemInput) error {
	reporter := report.R(ctx)
	config := LocalConfig{
		Dumper: &dumper.Configuraion{
			Compression:      dumper.Configuraion_PLANE,
			MaxDiskUsagePrct: 99,
		},
	}
	reporter.AddInt("signal", int64(si.Signal))

	pi, err := NewProcessInfo(ctx, si.InitialPid, si.NsPid, si.InitialTid, si.NsTid, afero.NewOsFs())
	if err != nil {
		return err
	}

	if si.PrGetDumpable.AllowCoreDump() {
		if _, err := dumper.New(afero.NewOsFs()).Dump(ctx, si.Stream, filepath.Join("/tmp", pi.CorefileName()), config.Dumper); err != nil {
			return err
		}
	} else {
		reporter.AddString("dump.status", "skipped")
	}
	return nil
}
