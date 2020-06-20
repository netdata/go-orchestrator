package build

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
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

type Registry interface {
	Register(name string) (bool, error)
	Unregister(name string) error
}

type (
	dummySaver    struct{}
	dummyState    struct{}
	dummyRegistry struct{}
)

func (d dummySaver) Save(_ confgroup.Config, _ string) {}
func (d dummySaver) Remove(_ confgroup.Config)         {}

func (d dummyState) Contains(_ confgroup.Config, _ ...string) bool { return false }

func (d dummyRegistry) Register(_ string) (bool, error) { return true, nil }
func (d dummyRegistry) Unregister(_ string) error       { return nil }

type state = string

const (
	success           state = "success"            // successfully started
	retry             state = "retry"              // failed, but we need keep trying auto-detection
	failed            state = "failed"             // failed
	duplicateLocal    state = "duplicate_local"    // a job with the same FullName is started
	duplicateGlobal   state = "duplicate_global"   // a job with the same FullName is registered
	registrationError state = "registration_error" // an error during registration
	buildError        state = "build_error"        // an error during building
)

type (
	Manager struct {
		PluginName string
		Out        io.Writer
		Modules    module.Registry
		*logger.Logger

		Runner    Runner
		CurState  StateSaver
		PrevState State
		Registry  Registry

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
		CurState:   dummySaver{},
		PrevState:  dummyState{},
		Registry:   dummyRegistry{},
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
	defer func() { m.cleanup(); m.Info("instance is stopped") }()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() { defer wg.Done(); m.runGroupProcessing(ctx, in) }()

	wg.Add(1)
	go func() { defer wg.Done(); m.runConfigProcessing(ctx) }()

	wg.Wait()
	<-ctx.Done()
}

func (m *Manager) cleanup() {
	for _, cancel := range *m.retryCache {
		cancel()
	}
	for name := range *m.startCache {
		_ = m.Registry.Unregister(name)
	}
}

func (m *Manager) runGroupProcessing(ctx context.Context, in <-chan []*confgroup.Group) {
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

func (m *Manager) runConfigProcessing(ctx context.Context) {
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
		m.Infof("module '%s' job '%s' is being served by another job, skipping it",
			cfg.Module(), cfg.Name())
		m.CurState.Save(cfg, duplicateLocal)
		return
	}

	cancel, isRetry := m.retryCache.lookup(cfg)
	if isRetry {
		cancel()
		m.retryCache.remove(cfg)
	}

	job, err := m.buildJob(cfg)
	if err != nil {
		m.Warningf("couldn't build module '%s' job '%s': %v", cfg.Module(), cfg.Name(), err)
		m.CurState.Save(cfg, buildError)
		return
	}

	if !isRetry && cfg.AutoDetectionRetry() == 0 {
		switch {
		case m.PrevState.Contains(cfg, success, retry):
			// TODO: method?
			// 5 minutes
			job.AutoDetectEvery = 30
			job.AutoDetectTries = 11
		case isInsideK8sCluster() && cfg.Provider() == "file watcher":
			// TODO: not sure this logic should belong to builder
			job.AutoDetectEvery = 10
			job.AutoDetectTries = 7
		}
	}

	switch detection(job) {
	case success:
		if ok, err := m.Registry.Register(cfg.FullName()); ok || err != nil && !isTooManyOpenFiles(err) {
			m.CurState.Save(cfg, success)
			m.Runner.Start(job)
			m.startCache.put(cfg)
		} else if isTooManyOpenFiles(err) {
			m.Error(err)
			m.CurState.Save(cfg, registrationError)
		} else {
			m.Infof("module '%s' job '%s'  is being served by another plugin, skipping it", cfg.Module(), cfg.Name())
			m.CurState.Save(cfg, duplicateGlobal)
		}
	case retry:
		m.Infof("module '%s' job '%s' detection failed, will retry in %d seconds",
			cfg.Module(), cfg.Name(), cfg.AutoDetectionRetry())
		m.CurState.Save(cfg, retry)
		ctx, cancel := context.WithCancel(ctx)
		m.retryCache.put(cfg, cancel)
		go retryTask(ctx, m.retryCh, cfg)
	case failed:
		m.CurState.Save(cfg, failed)
	default:
		m.Warningf("module '%s' job '%s' detection: unknown state '", cfg.Module(), cfg.Name())
	}
}

func (m *Manager) handleRemoveCfg(cfg confgroup.Config) {
	defer m.CurState.Remove(cfg)

	if m.startCache.has(cfg) {
		m.Runner.Stop(cfg.FullName())
		_ = m.Registry.Unregister(cfg.FullName())
		m.startCache.remove(cfg)
	}

	if cancel, ok := m.retryCache.lookup(cfg); ok {
		cancel()
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

func detection(job jobpkg.Job) state {
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

func isInsideK8sCluster() bool {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	return host != "" && port != ""
}

func isTooManyOpenFiles(err error) bool {
	return err != nil && strings.Contains(err.Error(), "too many open files")
}
