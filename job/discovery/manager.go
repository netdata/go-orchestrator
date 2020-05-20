package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/job/discovery/file"
)

type Config struct {
	Registry confgroup.Registry
	File     file.Config
}

func validateConfig(cfg Config) error {
	if len(cfg.Registry) == 0 {
		return errors.New("empty config registry")
	}
	if len(cfg.File.Dummy)+len(cfg.File.Read)+len(cfg.File.Watch) == 0 {
		return errors.New("empty config")
	}
	return nil
}

type (
	discoverer interface {
		Discover(ctx context.Context, in chan<- []*confgroup.Group)
	}
	Manager struct {
		discoverers []discoverer
		send        chan struct{}
		sendEvery   time.Duration
		mux         *sync.RWMutex
		cache       *cache
	}
)

func NewManager(cfg Config) (*Manager, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("discovery manager config validation: %v", err)
	}
	mgr := &Manager{
		send:        make(chan struct{}, 1),
		sendEvery:   time.Second,
		discoverers: make([]discoverer, 0),
		mux:         &sync.RWMutex{},
		cache:       newCache(),
	}
	if err := mgr.registerDiscoverers(cfg); err != nil {
		return nil, fmt.Errorf("discovery manager initializaion: %v", err)
	}
	return mgr, nil
}

func (m *Manager) registerDiscoverers(cfg Config) error {
	cfg.File.Registry = cfg.Registry
	d, err := file.NewDiscovery(cfg.File)
	if err != nil {
		return err
	}

	m.discoverers = append(m.discoverers, d)
	if len(m.discoverers) == 0 {
		return errors.New("zero registered discoverers")
	}
	return nil
}

func (m *Manager) Discover(ctx context.Context, in chan<- []*confgroup.Group) {
	var wg sync.WaitGroup

	for _, d := range m.discoverers {
		wg.Add(1)
		go func(d discoverer) {
			defer wg.Done()
			m.runDiscoverer(ctx, d)
		}(d)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.sendLoop(ctx, in)
	}()

	wg.Wait()
	<-ctx.Done()
}

func (m *Manager) runDiscoverer(ctx context.Context, d discoverer) {
	updates := make(chan []*confgroup.Group)
	go d.Discover(ctx, updates)

	for {
		select {
		case <-ctx.Done():
			return
		case groups, ok := <-updates:
			if !ok {
				return
			}
			func() {
				m.mux.Lock()
				defer m.mux.Unlock()

				m.cache.update(groups)
				m.triggerSend()
			}()
		}
	}
}

func (m *Manager) sendLoop(ctx context.Context, in chan<- []*confgroup.Group) {
	m.mustSend(ctx, in)

	tk := time.NewTicker(m.sendEvery)
	defer tk.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			select {
			case <-m.send:
				m.trySend(in)
			default:
			}
		}
	}
}

func (m *Manager) mustSend(ctx context.Context, in chan<- []*confgroup.Group) {
	select {
	case <-ctx.Done():
		return
	case <-m.send:
		m.mux.Lock()
		groups := m.cache.groups()
		m.cache.reset()
		m.mux.Unlock()

		select {
		case <-ctx.Done():
		case in <- groups:
		}
		return
	}
}

func (m *Manager) trySend(in chan<- []*confgroup.Group) {
	m.mux.Lock()
	defer m.mux.Unlock()

	select {
	case in <- m.cache.groups():
		m.cache.reset()
	default:
		m.triggerSend()
	}
}

func (m *Manager) triggerSend() {
	select {
	case m.send <- struct{}{}:
	default:
	}
}
