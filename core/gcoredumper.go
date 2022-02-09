package core

import (
	"syscall"
	"time"
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
}
