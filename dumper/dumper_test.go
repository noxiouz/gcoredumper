package dumper

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
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
			_, err := d.Dump(tc.ctx, bytes.NewBufferString(tc.filepath), tc.filepath, &Configuraion{
				Compression: Configuraion_PLANE,
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
		compression Configuraion_Compression
		want        []byte
	}{
		{
			name:        Configuraion_ZSTD.String(),
			compression: Configuraion_ZSTD,
			want: func() []byte {
				buff := bytes.NewBuffer(nil)
				enc, _ := zstd.NewWriter(buff)
				enc.Write(input)
				enc.Close()
				return buff.Bytes()
			}(),
		},
		{
			name:        Configuraion_SNAPPY.String(),
			compression: Configuraion_SNAPPY,
			want: func() []byte {
				buff := bytes.NewBuffer(nil)
				snappy.NewWriter(buff).Write(input)
				return buff.Bytes()
			}(),
		},
		{
			name:        Configuraion_PLANE.String(),
			compression: Configuraion_PLANE,
			want:        input,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			d := New(fs)
			f, err := d.Dump(ctx, bytes.NewReader(input), "/corefile1", &Configuraion{
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
