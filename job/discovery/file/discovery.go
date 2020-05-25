package file

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/netdata/go-orchestrator/job/confgroup"
)

type Config struct {
	Registry confgroup.Registry
	Dummy    []string
	Read     []string
	Watch    []string
}

func validateConfig(cfg Config) error {
	if len(cfg.Registry) == 0 {
		return errors.New("empty config registry")
	}
	if len(cfg.Dummy)+len(cfg.Read)+len(cfg.Watch) == 0 {
		return errors.New("empty config")
	}
	return nil
}

type (
	discoverer interface {
		Run(ctx context.Context, in chan<- []*confgroup.Group)
	}
	Discovery struct {
		req         confgroup.Registry
		discoverers []discoverer
	}
)

func NewDiscovery(cfg Config) (*Discovery, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("file discovery config validation: %v", err)
	}

	var d Discovery
	if err := d.registerDiscoverers(cfg); err != nil {
		return nil, fmt.Errorf("file discovery initialization: %v", err)
	}
	return &d, nil
}

func (d *Discovery) registerDiscoverers(cfg Config) error {
	if len(cfg.Dummy) != 0 {
		d.discoverers = append(d.discoverers, NewDummy(cfg.Registry, cfg.Dummy))
	}
	if len(cfg.Read) != 0 {
		d.discoverers = append(d.discoverers, NewReader(cfg.Registry, cfg.Read))
	}
	if len(cfg.Watch) != 0 {
		d.discoverers = append(d.discoverers, NewWatcher(cfg.Registry, cfg.Watch))
	}
	if len(d.discoverers) == 0 {
		return errors.New("zero registered discoverers")
	}
	return nil
}

func (d *Discovery) Run(ctx context.Context, in chan<- []*confgroup.Group) {
	var wg sync.WaitGroup

	for _, dd := range d.discoverers {
		wg.Add(1)
		go func(dd discoverer) {
			defer wg.Done()
			d.runDiscoverer(ctx, dd, in)
		}(dd)
	}

	wg.Wait()
	<-ctx.Done()
}

func (d *Discovery) runDiscoverer(ctx context.Context, dd discoverer, in chan<- []*confgroup.Group) {
	updates := make(chan []*confgroup.Group)
	go dd.Run(ctx, updates)
	for {
		select {
		case <-ctx.Done():
			return
		case groups, ok := <-updates:
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case in <- groups:
			}
		}
	}
}
