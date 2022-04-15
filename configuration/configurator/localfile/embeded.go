package localfile

import (
	"context"
	_ "embed"
	"errors"

	"github.com/noxiouz/gcoredumper/configuration"
	"github.com/noxiouz/gcoredumper/configuration/configurator"
	"google.golang.org/protobuf/encoding/prototext"
)

func init() {
	configurator.Register("embed", configurator.FactoryFunc(NewEmbedded))
}

//go:embed config_samle.prototxt
var configSample []byte

type Embedded struct{}

func (Embedded) Get(ctx context.Context) (*configuration.Config, error) {
	m := new(configuration.Config)
	if err := prototext.Unmarshal(configSample, m); err != nil {
		return nil, err
	}
	return m, nil
}

func NewEmbedded(path string) (configurator.Configurator, error) {
	if len(configSample) == 0 {
		return nil, errors.New("embedded config is empty")
	}
	return Embedded{}, nil
}
