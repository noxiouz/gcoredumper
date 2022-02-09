package environ

import (
	"bufio"
	"bytes"
	"io"
)

type Environ interface {
	GetVar(string) string
	HasVar(string) bool
}

type mapEnviron map[string]string

func (m mapEnviron) GetVar(v string) string {
	return m[v]
}

func (m mapEnviron) HasVar(v string) bool {
	_, ok := m[v]
	return ok
}

// Parses a new environ from null-terminated list of strings. Like /proc/<pid>/environ
func New(r io.Reader) Environ {
	res := make(map[string]string)
	scanner := bufio.NewScanner(r)
	scanner.Split(scanNullByte)
	for scanner.Scan() {
		kv := bytes.SplitN(scanner.Bytes(), []byte("="), 2)
		switch len(kv) {
		case 2:
			res[string(kv[0])] = string(kv[1])
		case 1:
			res[string(kv[0])] = ""
		}
	}
	return mapEnviron(res)
}

func scanNullByte(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\x00'); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
