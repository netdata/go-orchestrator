package plugin

import (
	"context"
	"errors"
	"fmt"
	"github.com/netdata/go-orchestrator/job/build"
	"github.com/netdata/go-orchestrator/job/run"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/job/discovery"
	"github.com/netdata/go-orchestrator/job/discovery/file"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/logger"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"github.com/mattn/go-isatty"
	"gopkg.in/yaml.v2"
)

var (
	log = logger.New("plugin", "main", "main")
)

var isTerminal = isatty.IsTerminal(os.Stdout.Fd())

type Config struct {
	Name           string
	MinUpdateEvery int
	UseModule      string
	ConfDir        []string
	SDConfPath     []string
}

// Plugin represents orchestrator.
type Plugin struct {
	Name           string
	Out            io.Writer
	Registry       module.Registry
	ConfDir        multipath.MultiPath
	SDConfPath     []string
	MinUpdateEvery int

	UseModule      string
	enabledModules module.Registry
}

// New creates Plugin.
func New(cfg Config) (*Plugin, error) {
	p := &Plugin{
		Name: "go.d",
		ConfDir: multipath.New(
			os.Getenv("NETDATA_USER_CONFIG_DIR"),
			os.Getenv("NETDATA_STOCK_CONFIG_DIR"),
		),
		Registry:       module.DefaultRegistry,
		Out:            os.Stdout,
		MinUpdateEvery: module.UpdateEvery,
		enabledModules: make(module.Registry),
	}

	if cfg.Name != "" {
		p.Name = cfg.Name
	}
	if cfg.MinUpdateEvery > 0 {
		p.MinUpdateEvery = cfg.MinUpdateEvery
	}
	if cfg.UseModule != "" {
		p.UseModule = cfg.UseModule
	}
	if len(cfg.ConfDir) > 0 {
		p.ConfDir = multipath.New(cfg.ConfDir...)
	}
	if len(cfg.SDConfPath) > 0 {
		p.SDConfPath = cfg.SDConfPath
	}

	if !isTerminal {
		logger.SetPluginName(p.Name, log)
	}

	pluginCfg, err := p.loadConfig()
	if err != nil {
		return nil, err
	}

	p.loadModules(*pluginCfg)
	if len(p.enabledModules) == 0 {
		return nil, errors.New("no modules to run")
	}

	if pluginCfg.MaxProcs > 0 {
		runtime.GOMAXPROCS(pluginCfg.MaxProcs)
	}

	log.Infof("maximum number of used CPUs %d", pluginCfg.MaxProcs)
	log.Infof("minimum update every %d", p.MinUpdateEvery)

	return p, nil
}

// Run
func (p *Plugin) Run() {
	go signalHandling()

	if !isTerminal {
		go keepAlive()
	}

	dm, err := p.initDiscoveryManager()
	if err != nil {
		panic(err)
	}

	bm := build.NewManager()
	rm := run.NewManager()
	bm.Discoverer = dm
	bm.Runner = rm
	bm.Modules = p.enabledModules
	bm.Out = p.Out
	bm.PluginName = p.Name

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() { defer wg.Done(); rm.Run(ctx) }()

	wg.Add(1)
	go func() { defer wg.Done(); bm.Run(ctx) }()

	wg.Wait()
	<-ctx.Done()
}

func (p *Plugin) initDiscoveryManager() (*discovery.Manager, error) {
	var paths, dummyPaths []string
	reg := confgroup.Registry{}

	for name, creator := range p.enabledModules {
		reg.Register(name, confgroup.Default{
			MinUpdateEvery:     p.MinUpdateEvery,
			UpdateEvery:        creator.UpdateEvery,
			AutoDetectionRetry: creator.AutoDetectionRetry,
			Priority:           creator.Priority,
		})

		path, err := p.ConfDir.Find(name + ".conf")
		if err != nil && !multipath.IsNotFound(err) {
			continue
		}

		if err != nil {
			//dummyPaths = append(dummyPaths, name)
		} else {
			paths = append(paths, path)
		}
	}

	return discovery.NewManager(discovery.Config{
		Registry: reg,
		File: file.Config{
			Dummy: dummyPaths,
			Read:  paths,
			Watch: p.SDConfPath,
		},
	})
}

func signalHandling() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGPIPE)

	sig := <-sigCh
	log.Infof("received %s signal (%d). Terminating...", sig, sig)

	switch sig {
	case syscall.SIGPIPE:
		os.Exit(1)
	default:
		os.Exit(0)
	}
}

func keepAlive() {
	t := time.Tick(time.Second)
	for range t {
		_, _ = fmt.Fprint(os.Stdout, "\n")
	}
}

func (p *Plugin) loadConfig() (*config, error) {
	path, err := p.ConfDir.Find(p.Name + ".conf")

	if err != nil && !multipath.IsNotFound(err) {
		return nil, fmt.Errorf("find configuration file: %v", err)
	}

	cfg := config{
		Enabled:    true,
		DefaultRun: true,
		MaxProcs:   0,
		Modules:    nil,
	}

	if err != nil && multipath.IsNotFound(err) {
		log.Warningf("find configuration file: %v, will use defaults", err)
		return &cfg, nil
	}

	if err := loadYAML(&cfg, path); err != nil {
		return nil, fmt.Errorf("load configuration: %v", err)
	}
	return &cfg, nil
}

func (p *Plugin) loadModules(cfg config) {
	all := p.UseModule == "all" || p.UseModule == ""

	for name, creator := range p.Registry {
		if !all && p.UseModule != name {
			continue
		}
		if all && creator.Disabled && !cfg.isModuleExplicitlyEnabled(name) {
			log.Infof("module '%s' disabled by default", name)
			continue
		}
		if all && !cfg.isModuleImplicitlyEnabled(name) {
			log.Infof("module '%s' disabled in configuration file", name)
			continue
		}
		p.enabledModules[name] = creator
	}
}

func loadYAML(conf interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = yaml.NewDecoder(f).Decode(conf); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return nil
}
