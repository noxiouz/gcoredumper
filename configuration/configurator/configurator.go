package configurator

import (
	"context"
	"fmt"
	"sync"

	"github.com/noxiouz/gcoredumper/configuration"
)

type Factory interface {
	Open(path string) (Configurator, error)
}

type FactoryFunc func(path string) (Configurator, error)

func (f FactoryFunc) Open(path string) (Configurator, error) {
	return f(path)
}

var (
	mu        sync.Mutex
	factories = make(map[string]Factory)
)

func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = factory
}

type Configurator interface {
	Get(ctx context.Context) (*configuration.Config, error)
}

func Open(name, path string) (Configurator, error) {
	mu.Lock()
	defer mu.Unlock()

	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("unknown factory %s", name)
	}
	return factory.Open(path)
}
