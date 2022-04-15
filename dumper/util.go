package dumper

import "io"

// Wraps a file with no-op Close method
type writerNopCloser struct {
	io.Writer
}

// Close intentionally does nothing
func (writerNopCloser) Close() error {
	return nil
}
