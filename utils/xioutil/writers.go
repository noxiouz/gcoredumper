package xioutil

import (
	"context"
	"io"
)

type WhileFunc func([]byte) error

type whileWriter struct {
	fn WhileFunc
	wr io.Writer
}

func NewWhileWriter(fn WhileFunc, wr io.Writer) io.Writer {
	return &whileWriter{
		fn: fn,
		wr: wr,
	}
}

func (ww *whileWriter) Write(p []byte) (int, error) {
	if err := ww.fn(p); err != nil {
		return 0, err
	}
	return ww.wr.Write(p)
}

func NewCancellableWriter(ctx context.Context, wr io.Writer) io.Writer {
	fn := func([]byte) error {
		return ctx.Err()
	}
	return NewWhileWriter(fn, wr)
}
