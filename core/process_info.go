package core

import (
	"context"
	"debug/elf"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/noxiouz/gcoredumper/report"
	"github.com/noxiouz/gcoredumper/utils/buildid"
	"github.com/noxiouz/gcoredumper/utils/environ"
)

type ProcessInfo interface {
	HasPIDNamespace() bool
	IsBinaryDeleted() bool
	Env() environ.Environ
	CorefileName() string
}

type processInfoImpl struct {
	// pid in a global/local namespace
	globalPid int64
	localPid  int64
	// tid of a crashed thread in a global/local namespace
	globalTid int64
	localTid  int64

	cmdline string
	// executable
	excutable         string
	binary            string
	executableDeleted bool
	//
	env environ.Environ
}

const (
	deletedBinarySuffix = "(deleted)"
)

func NewProcessInfo(ctx context.Context, globalPid int64, localPid int64, globalTid int64, localTid int64, filesystem afero.Fs) (ProcessInfo, error) {
	pi := processInfoImpl{
		globalPid: globalPid,
		localPid:  localPid,
		globalTid: globalTid,
		localTid:  localTid,
	}

	report.R(ctx).AddInt("pid.global", globalPid)
	report.R(ctx).AddInt("pid.ns", localPid)
	report.R(ctx).AddInt("tid.global", globalTid)
	report.R(ctx).AddInt("tid.ns", localTid)

	procFs := afero.NewBasePathFs(filesystem, fmt.Sprintf("/proc/%d", globalPid))
	cmdline, err := afero.ReadFile(procFs, "cmdline")
	if err != nil {
		return nil, err
	}
	pi.cmdline = string(cmdline)
	report.R(ctx).AddString("cmdline", pi.cmdline)

	// ignore error. fs.FS does not support Readlink so do it directly
	if linkReader, ok := procFs.(afero.LinkReader); ok {
		pi.excutable, _ = linkReader.ReadlinkIfPossible("exe")
		pi.executableDeleted = strings.HasSuffix(pi.excutable, deletedBinarySuffix)
		pi.excutable = strings.TrimSuffix(pi.excutable, deletedBinarySuffix)
	}
	pi.binary = filepath.Base(pi.excutable)
	report.R(ctx).AddString("binary", pi.binary)

	if err := extractElfInfo(procFs, report.R(ctx)); err != nil {
		return nil, err
	}

	// Read environment vars
	environFile, err := procFs.Open("environ")
	if err != nil {
		return nil, err
	}
	defer environFile.Close()
	pi.env = environ.New(environFile)
	return &pi, nil
}

func (p *processInfoImpl) HasPIDNamespace() bool {
	return p.globalPid == p.localPid
}

func (p *processInfoImpl) IsBinaryDeleted() bool {
	return p.executableDeleted
}

func (p *processInfoImpl) Env() environ.Environ {
	return p.env
}

func (p *processInfoImpl) CorefileName() string {
	return fmt.Sprintf("%s.%d.%d", p.binary, p.globalPid, p.globalTid)
}

func extractElfInfo(procFs afero.Fs, rep *report.Report) error {
	exe, err := procFs.Open("exe")
	if err != nil {
		return err
	}
	defer exe.Close()

	ef, err := elf.NewFile(exe)
	if err != nil {
		return err
	}

	buildId, err := buildid.New(ef)
	if err != nil {
		return err
	}

	rep.AddString("binary.buildid", buildId)
	return nil
}
