package core

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/noxiouz/gcoredumper/report"
)

type ActionFunc func(context.Context, *ProcessInfo) error

func (af ActionFunc) Run(ctx context.Context, pinfo *ProcessInfo) error {
	return af(ctx, pinfo)
}

const (
	dpkgExe      = "/usr/bin/dpkg"
	dpkgQueryExe = "/usr/bin/dpkg-query"

	rpmExe = "/usr/bin/rpm"
)

func PackageVersionAction(ctx context.Context, pinfo *ProcessInfo) error {
	// TODO: skip if binary is inside containter (mount ns?)
	log.Printf("Deteting package name and version for %s", pinfo.excutable)
	fileExists := func(file string) bool {
		_, err := os.Stat(file)
		return err == nil
	}
	var (
		nameAndVersion string
		err            error
	)
	switch {
	case fileExists(dpkgExe):
		nameAndVersion, err = dpkgPackageVersionAction(ctx, pinfo)
		if err != nil {
			log.Println(err)
			return nil // Ignore error, not critical
		}
	case fileExists(rpmExe):
		nameAndVersion, err = rpmPackageVersionAction(ctx, pinfo)
		if err != nil {
			log.Println(err)
			return nil // Ignore error, not critical
		}
	default:
		log.Println("Package detection is not supported for the system")
		return nil
	}
	report.R(ctx).AddString("package.name", nameAndVersion)
	return nil
}

func dpkgPackageVersionAction(ctx context.Context, pinfo *ProcessInfo) (string, error) {
	nobodyCredentials, err := getNobodyCredentials()
	if err != nil {
		return "", err
	}
	// NOTE: $PATH is empty. Specify absilute path to an executable.
	cmd := exec.CommandContext(ctx, dpkgExe, "-S", pinfo.excutable)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:  syscall.SIGKILL,
		Credential: nobodyCredentials,
	}
	output, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			log.Printf("%s", err.Stderr)
		}
		return "", err
	}
	log.Printf("dpkg -S %s: %s", pinfo.excutable, output)

	delimPos := bytes.IndexByte(output, ':')
	if delimPos == -1 {
		return "", fmt.Errorf("dpkg -S output is malformed")
	}

	dpkgQueryArgs := []string{
		"-W", "-f='${Package}-${Version}'", string(output[:delimPos]),
	}
	cmd = exec.CommandContext(ctx, dpkgQueryExe, dpkgQueryArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:  syscall.SIGKILL,
		Credential: nobodyCredentials,
	}
	nameAndVersion, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(nameAndVersion), nil
}

func rpmPackageVersionAction(ctx context.Context, pinfo *ProcessInfo) (string, error) {
	nobodyCredentials, err := getNobodyCredentials()
	if err != nil {
		return "", err
	}
	rpmQueryArgs := []string{
		"-qf", pinfo.excutable,
	}
	cmd := exec.CommandContext(ctx, rpmExe, rpmQueryArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:  syscall.SIGKILL,
		Credential: nobodyCredentials,
	}
	nameAndVersion, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(nameAndVersion), nil
}

func getNobodyCredentials() (*syscall.Credential, error) {
	nobody, err := user.Lookup("nobody")
	if err != nil {
		return nil, err
	}
	nobodyUid, err := strconv.Atoi(nobody.Uid)
	if err != nil {
		return nil, err
	}
	nobodyGid, err := strconv.Atoi(nobody.Gid)
	if err != nil {
		return nil, err
	}

	return &syscall.Credential{
		Uid: uint32(nobodyUid),
		Gid: uint32(nobodyGid),
	}, nil
}
