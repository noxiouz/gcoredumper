package buildid

import (
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
)

// ErrNoBuildId is returned when Build Id has not been detected by the library.
var ErrNoBuildId = errors.New("No BuildID detected")

type buildId []byte

func New(f *elf.File) (string, error) {
	var bid buildId
	for _, section := range f.Sections {
		switch section.Name {
		case ".note.gnu.build-id":
			if err := bid.UnmarshalBinary(section.Open()); err != nil {
				return "", err
			}
			return hex.EncodeToString(bid), nil
		case ".note.go.buildid": // Go binary
			if err := bid.UnmarshalBinary(section.Open()); err != nil {
				return "", err
			}
			return string(bid), nil
		}
	}
	return "", ErrNoBuildId
}

func (b *buildId) UnmarshalBinary(r io.Reader) error {
	var buildIDHeader struct {
		NameSz uint32
		DescSz uint32
		Type   uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &buildIDHeader); err != nil {
		return err
	}
	name := make([]byte, buildIDHeader.NameSz)
	if err := binary.Read(r, binary.LittleEndian, name); err != nil {
		return err
	}
	id := make([]byte, buildIDHeader.DescSz)
	if err := binary.Read(r, binary.LittleEndian, id); err != nil {
		return err
	}
	*b = buildId(id)
	return nil
}
