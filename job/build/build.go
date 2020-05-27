package build

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	jobpkg "github.com/netdata/go-orchestrator/job"
	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/logger"

	"gopkg.in/yaml.v2"
)

type Runner interface {
	Start(job jobpkg.Job)
	Stop(fullName string)
}

type StateSaver interface {
	Save(cfg confgroup.Config, state string)
	Remove(cfg confgroup.Config)
}

type State interface {
	Contains(cfg confgroup.Config, states ...string) bool
}

type (
	dummySaver struct{}
	dummyState struct{}
)

func (d dummySaver) Save(cfg confgroup.Config, state string)              {}
func (d dummySaver) Remove(cfg confgroup.Config)                          {}
func (d dummyState) Contains(cfg confgroup.Config, states ...string) bool { return false }

type state = string

const (
	success    state = "success"     // successfully started
	retry      state = "retry"       // failed, but we need keep trying auto-detection
	failed     state = "failed"      // failed
	duplicate  state = "duplicate"   // there is already 'success' job with the same FullName
	buildError state = "build_error" // error during building
)

type (
	Manager struct {
		PluginName string
		Out        io.Writer
		Modules    module.Registry
		*logger.Logger

		Runner    Runner
		Saver     StateSaver
		PrevState State

		grpCache   *groupCache
		startCache *startedCache
		retryCache *retryCache

		addCh    chan []confgroup.Config
		removeCh chan []confgroup.Config
		retryCh  chan confgroup.Config
	}
)

func NewManager() *Manager {
	mgr := &Manager{
		Saver:      dummySaver{},
		PrevState:  dummyState{},
		Out:        ioutil.Discard,
		Logger:     logger.New("build", "manager"),
		grpCache:   newGroupCache(),
		startCache: newStartedCache(),
		retryCache: newRetryCache(),
		addCh:      make(chan []confgroup.Config),
		removeCh:   make(chan []confgroup.Config),
		retryCh:    make(chan confgroup.Config),
	}
	return mgr
}

func (m *Manager) Run(ctx context.Context, in chan []*confgroup.Group) {
	m.Info("instance is started")
	defer func() { m.Info("instance is stopped") }()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() { defer wg.Done(); m.runProcessing(ctx, in) }()

	wg.Add(1)
	go func() { defer wg.Done(); m.runHandleEvents(ctx) }()

	wg.Wait()
	<-ctx.Done()
}

func (m *Manager) runProcessing(ctx context.Context, in <-chan []*confgroup.Group) {
	for {
		select {
		case <-ctx.Done():
			return
		case groups := <-in:
			for _, group := range groups {
				select {
				case <-ctx.Done():
					return
				default:
					m.processGroup(ctx, group)
				}
			}
		}
	}
}

func (m *Manager) processGroup(ctx context.Context, group *confgroup.Group) {
	if group == nil {
		return
	}
	added, removed := m.grpCache.put(group)

	select {
	case <-ctx.Done():
		return
	case m.removeCh <- removed:
	}

	select {
	case <-ctx.Done():
		return
	case m.addCh <- added:
	}
}

func (m *Manager) runHandleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cfgs := <-m.addCh:
			m.handleAdd(ctx, cfgs)
		case cfgs := <-m.removeCh:
			m.handleRemove(ctx, cfgs)
		case cfg := <-m.retryCh:
			m.handleAddCfg(ctx, cfg)
		}
	}
}

func (m *Manager) handleAdd(ctx context.Context, cfgs []confgroup.Config) {
	for _, cfg := range cfgs {
		select {
		case <-ctx.Done():
			return
		default:
			m.handleAddCfg(ctx, cfg)
		}
	}
}

func (m *Manager) handleRemove(ctx context.Context, cfgs []confgroup.Config) {
	for _, cfg := range cfgs {
		select {
		case <-ctx.Done():
			return
		default:
			m.handleRemoveCfg(cfg)
		}
	}
}

func (m *Manager) handleAddCfg(ctx context.Context, cfg confgroup.Config) {
	if m.startCache.has(cfg) {
		m.Saver.Save(cfg, duplicate)
		return
	}

	stop, isRetry := m.retryCache.lookup(cfg)
	if isRetry {
		stop()
		m.retryCache.remove(cfg)
	}

	job, err := m.buildJob(cfg)
	if err != nil {
		m.Warningf("couldn't build job: %v", err)
		m.Saver.Save(cfg, buildError)
		return
	}

	if !isRetry && m.PrevState.Contains(cfg, success, retry) {
		// TODO: method?
		// 5 minutes
		job.AutoDetectEvery = 30
		job.AutoDetectTries = 11
	}

	switch v := runDetection(job); v {
	case success:
		m.Saver.Save(cfg, success)
		m.Runner.Start(job)
		m.startCache.put(cfg)
	case retry:
		m.Saver.Save(cfg, retry)
		ctx, cancel := context.WithCancel(ctx)
		m.retryCache.put(cfg, cancel)
		go retryTask(ctx, m.retryCh, cfg)
	case failed:
		m.Saver.Save(cfg, failed)
	default:
		m.Warningf("unknown detection state: '%s', module '%s' job '%s'", v, cfg.Module(), cfg.Name())
	}
}

func (m *Manager) handleRemoveCfg(cfg confgroup.Config) {
	defer m.Saver.Remove(cfg)
	if m.startCache.has(cfg) {
		m.Runner.Stop(cfg.FullName())
		m.startCache.remove(cfg)
		return
	}

	if stop, ok := m.retryCache.lookup(cfg); ok {
		stop()
		m.retryCache.remove(cfg)
	}
}

func (m *Manager) buildJob(cfg confgroup.Config) (*module.Job, error) {
	creator, ok := m.Modules[cfg.Module()]
	if !ok {
		return nil, fmt.Errorf("couldn't find '%s' module, job '%s'", cfg.Module(), cfg.Name())
	}

	mod := creator.Create()
	if err := unmarshal(cfg, mod); err != nil {
		return nil, err
	}

	job := module.NewJob(module.JobConfig{
		PluginName:      m.PluginName,
		Name:            cfg.Name(),
		ModuleName:      cfg.Module(),
		FullName:        cfg.FullName(),
		UpdateEvery:     cfg.UpdateEvery(),
		AutoDetectEvery: cfg.AutoDetectionRetry(),
		Priority:        cfg.Priority(),
		Module:          mod,
		Out:             m.Out,
	})
	return job, nil
}

func runDetection(job jobpkg.Job) state {
	if !job.AutoDetection() {
		if job.RetryAutoDetection() {
			return retry
		} else {
			return failed
		}
	}
	return success
}

func retryTask(ctx context.Context, in chan<- confgroup.Config, cfg confgroup.Config) {
	timeout := time.Second * time.Duration(cfg.AutoDetectionRetry())
	t := time.NewTimer(timeout)
	defer t.Stop()

	select {
	case <-ctx.Done():
	case <-t.C:
		select {
		case <-ctx.Done():
		case in <- cfg:
		}
	}
}

func unmarshal(conf interface{}, module interface{}) error {
	bs, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bs, module)
}
