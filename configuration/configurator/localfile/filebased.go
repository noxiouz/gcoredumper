package localfile

import (
	"context"
	"os"

	"github.com/noxiouz/gcoredumper/configuration"
	"github.com/noxiouz/gcoredumper/configuration/configurator"
	"google.golang.org/protobuf/encoding/prototext"
)

func init() {
	configurator.Register("file", configurator.FactoryFunc(NewFileBased))
}

type fileConfigurator struct {
	path string
}

func (f fileConfigurator) Get(ctx context.Context) (*configuration.Config, error) {
	body, err := os.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	m := new(configuration.Config)
	if err := prototext.Unmarshal(body, m); err != nil {
		return nil, err
	}
	return m, nil
}

func NewFileBased(path string) (configurator.Configurator, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return fileConfigurator{path}, nil
}
