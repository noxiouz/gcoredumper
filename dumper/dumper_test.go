package dumper

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/noxiouz/gcoredumper/configuration"
	"github.com/spf13/afero"
)

func TestDumpErrors(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		ctx      context.Context
		filepath string
	}{
		{
			name: "ContextCancellation",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			}(),
			filepath: "/corefile1",
		},
		{
			name:     "NoDirectory",
			ctx:      ctx,
			filepath: "/nodir/corefile1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := New(afero.NewMemMapFs())
			_, err := d.Dump(tc.ctx, bytes.NewBufferString(tc.filepath), tc.filepath, &configuration.Config_DumperConfig{
				Compression: configuration.Config_DumperConfig_PLANE,
			})
			if err == nil { // if NO error
				t.Errorf("Dump() expected to return an error, but got nil")
			}
		})
	}
}

func TestDumpWithCompression(t *testing.T) {
	ctx := context.Background()
	input := []byte("some_non_random_content")
	for _, tc := range []struct {
		name        string
		compression configuration.Config_DumperConfig_Compression
		want        []byte
	}{
		{
			name:        configuration.Config_DumperConfig_ZSTD.String(),
			compression: configuration.Config_DumperConfig_ZSTD,
			want: func() []byte {
				buff := bytes.NewBuffer(nil)
				enc, _ := zstd.NewWriter(buff)
				enc.Write(input)
				enc.Close()
				return buff.Bytes()
			}(),
		},
		{
			name:        configuration.Config_DumperConfig_SNAPPY.String(),
			compression: configuration.Config_DumperConfig_SNAPPY,
			want: func() []byte {
				buff := bytes.NewBuffer(nil)
				snappy.NewWriter(buff).Write(input)
				return buff.Bytes()
			}(),
		},
		{
			name:        configuration.Config_DumperConfig_PLANE.String(),
			compression: configuration.Config_DumperConfig_PLANE,
			want:        input,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			d := New(fs)
			f, err := d.Dump(ctx, bytes.NewReader(input), "/corefile1", &configuration.Config_DumperConfig{
				Compression: tc.compression,
			})
			if err != nil {
				t.Fatalf("Dump returned unexpected error %v", err)
			}
			got, err := afero.ReadFile(fs, f)
			if err != nil {
				t.Fatalf("ReadFile(%s) returned an error %v", f, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ReadFile() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
