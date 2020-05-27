package plugin

import (
	"context"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/netdata/go-orchestrator/job/build"
	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/job/discovery"
	"github.com/netdata/go-orchestrator/job/run"
	"github.com/netdata/go-orchestrator/job/state"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/logger"
	"github.com/netdata/go-orchestrator/pkg/multipath"
	"github.com/netdata/go-orchestrator/pkg/netdataapi"

	"github.com/mattn/go-isatty"
)

//var log = logger.New("plugin", "main", "main")

var isTerminal = isatty.IsTerminal(os.Stdout.Fd())

// Config is Plugin configuration.
type Config struct {
	Name               string
	ConfPath           []string
	ModulesConfPath    []string
	ModulesSDConfFiles []string
	StateFile          string
	ModuleRegistry     module.Registry
	RunModule          string
	MinUpdateEvery     int
}

// Plugin represents orchestrator.
type Plugin struct {
	Name              string
	ConfDir           multipath.MultiPath
	ModulesConfDir    multipath.MultiPath
	ModulesSDConfPath []string
	StateFile         string
	RunModule         string
	MinUpdateEvery    int
	ModuleRegistry    module.Registry
	Out               io.Writer
	*logger.Logger
}

// New creates a new Plugin.
func New(cfg Config) *Plugin {
	p := &Plugin{
		Name:              cfg.Name,
		ConfDir:           cfg.ConfPath,
		ModulesConfDir:    cfg.ModulesConfPath,
		ModulesSDConfPath: cfg.ModulesSDConfFiles,
		RunModule:         cfg.RunModule,
		MinUpdateEvery:    cfg.MinUpdateEvery,
		ModuleRegistry:    module.DefaultRegistry,
		Out:               os.Stdout,
	}

	logger.Prefix = p.Name
	p.Logger = logger.NewNamed("main", "main")

	return p
}

// Run
func (p *Plugin) Run() {
	p.Info("starting...")
	go p.signalHandling()
	go p.keepAlive()
	serve(p)
}

func serve(p *Plugin) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	handleReload(ch, p)
}

func handleReload(ch chan os.Signal, p *Plugin) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() { defer wg.Done(); p.run(ctx) }()

	<-ch
	cancel()
	wg.Wait()
	handleReload(ch, p)
}

func (p *Plugin) run(ctx context.Context) {
	cfg := p.loadPluginConfig()
	if !cfg.Enabled {
		p.Info("plugin is disabled in the configuration file, exiting...")
		if !isTerminal {
			p.disable()
		}
		os.Exit(0)
	}

	enabled := p.loadEnabledModules(cfg)
	if len(enabled) == 0 {
		p.Info("no modules to run")
		if isTerminal {
			os.Exit(0)
		}
		return
	}

	discCfg := p.buildDiscoveryConf(enabled)

	discoverer, err := discovery.NewManager(discCfg)
	if err != nil {
		if isTerminal {
			os.Exit(0)
		}
		return
	}

	runner := run.NewManager()

	builder := build.NewManager()
	builder.Runner = runner
	builder.Modules = enabled
	builder.Out = p.Out

	var saver *state.Manager
	if !isTerminal && p.StateFile != "" {
		saver = state.NewManager(p.StateFile)
		builder.Saver = saver
		st, err := state.Load(p.StateFile)
		if err != nil {
			builder.PrevState = st
		}
	}

	in := make(chan []*confgroup.Group)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() { defer wg.Done(); runner.Run(ctx) }()

	wg.Add(1)
	go func() { defer wg.Done(); builder.Run(ctx, in) }()

	wg.Add(1)
	go func() { defer wg.Done(); discoverer.Run(ctx, in) }()

	if saver != nil {
		wg.Add(1)
		go func() { defer wg.Done(); saver.Run(ctx) }()
	}

	wg.Wait()
	<-ctx.Done()
	runner.Cleanup()
}

func (p *Plugin) disable() {
	_ = netdataapi.New(p.Out).DISABLE()
}

func (p *Plugin) signalHandling() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)

	sig := <-ch
	p.Infof("received %s signal (%d). Terminating...", sig, sig)

	switch sig {
	case syscall.SIGPIPE:
		os.Exit(1)
	default:
		os.Exit(0)
	}
}

func (p *Plugin) keepAlive() {
	if isTerminal {
		return
	}
	api := netdataapi.New(p.Out)
	for range time.Tick(time.Second) {
		_ = api.EMPTYLINE()
	}
}
