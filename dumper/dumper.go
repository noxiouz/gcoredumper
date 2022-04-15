package dumper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"golang.org/x/sys/unix"

	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/afero"

	"github.com/noxiouz/gcoredumper/configuration"
	"github.com/noxiouz/gcoredumper/report"
	"github.com/noxiouz/gcoredumper/utils/xioutil"
)

type Dumper struct {
	fs afero.Fs
}

func New(fs afero.Fs) *Dumper {
	return &Dumper{
		fs: fs,
	}
}

func (d *Dumper) Dump(ctx context.Context, r io.Reader, filepath string, config *configuration.Config_DumperConfig) (string, error) {
	reporter := report.R(ctx)
	dumpStarted := time.Now()

	directory := path.Dir(filepath)
	if exists, err := afero.DirExists(d.fs, directory); err != nil {
		return "", err
	} else if !exists {
		return "", fmt.Errorf("%s does not exist", directory)
	}
	if suffix := getCorefileSuffix(config.GetCompression()); suffix != "" {
		filepath = filepath + suffix
	}

	file, err := d.fs.Create(path.Join(filepath))
	if err != nil {
		return "", err
	}
	defer file.Close()

	compressor, err := newCompressor(config, file)
	if err != nil {
		return "", err
	}
	defer compressor.Close()

	log.Printf("a coredump will be stored to %s", filepath)
	log.Printf("a coredumper will be compressed with %s", config.Compression)

	wr := xioutil.NewCancellableWriter(ctx, compressor)
	if osFile, ok := file.(*os.File); ok {
		diskUsageFn := func(p []byte) error {
			var stat unix.Statfs_t
			if err := unix.Fstatfs(int(osFile.Fd()), &stat); err != nil {
				return err
			}

			blocksUsed := stat.Blocks - stat.Bavail // exclude Bfree
			usagePct := uint(float64(blocksUsed) / float64(stat.Blocks) * 100)
			if usagePct > uint(config.MaxDiskUsagePrct) {
				return errors.New("no enough space")
			}
			return nil
		}
		wr = xioutil.NewWhileWriter(diskUsageFn, wr)
	}
	coreSize, err := io.CopyBuffer(wr, r, nil)
	if err != nil {
		os.Remove(file.Name())
	}

	if err == nil { // if NO error
		reporter.AddInt("core.size", coreSize)
		reporter.AddString("core.filepath", filepath)
		reporter.AddDuration("core.dumpingduration", time.Now().Sub(dumpStarted))
	}
	reporter.AddError("core.error", err)
	return filepath, err
}

func newCompressor(cfg *configuration.Config_DumperConfig, wr io.Writer) (io.WriteCloser, error) {
	switch compression := cfg.GetCompression(); compression {
	case configuration.Config_DumperConfig_PLANE:
		return writerNopCloser{wr}, nil
	case configuration.Config_DumperConfig_ZSTD:
		return zstd.NewWriter(wr)
	case configuration.Config_DumperConfig_SNAPPY:
		return snappy.NewBufferedWriter(wr), nil
	default:
		return nil, fmt.Errorf("unknown Compression type %d", compression)
	}
}

func getCorefileSuffix(compression configuration.Config_DumperConfig_Compression) string {
	switch compression {
	case configuration.Config_DumperConfig_PLANE:
		return ""
	case configuration.Config_DumperConfig_ZSTD:
		return ".zstd"
	case configuration.Config_DumperConfig_SNAPPY:
		return ".snappy"
	default:
		return ""
	}
}
