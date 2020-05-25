package plugin

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/logger"
	"github.com/netdata/go-orchestrator/pkg/multipath"
	"github.com/netdata/go-orchestrator/pkg/netdataapi"

	"github.com/mattn/go-isatty"
)

var varLibDir = os.Getenv("NETDATA_LIB_DIR")

const stateFile = "god-jobs-statuses.json"

var (
	log = logger.New("plugin", "main", "main")
)

var isTerminal = isatty.IsTerminal(os.Stdout.Fd())

/*
	PluginConfPath: multipath.New(
		os.Getenv("NETDATA_USER_CONFIG_DIR"),
		os.Getenv("NETDATA_STOCK_CONFIG_DIR"),
	),
	ModulesConfPath: multipath.New(
		filepath.Join(os.Getenv("NETDATA_USER_CONFIG_DIR"), "go.d"),
		filepath.Join(os.Getenv("NETDATA_STOCK_CONFIG_DIR"), "go.d"),
	),
	ModulesSDConfFiles: nil,
*/

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
	Name               string
	ConfPath           multipath.MultiPath
	ModulesConfPath    multipath.MultiPath
	ModulesSDConfFiles []string
	StateFile          string
	RunModule          string
	MinUpdateEvery     int
	ModuleRegistry     module.Registry
	Out                io.Writer
}

// New creates Plugin.
func New(cfg Config) *Plugin {
	return &Plugin{
		Name:               cfg.Name,
		ConfPath:           cfg.ConfPath,
		ModulesConfPath:    cfg.ModulesConfPath,
		ModulesSDConfFiles: cfg.ModulesSDConfFiles,
		RunModule:          cfg.RunModule,
		MinUpdateEvery:     cfg.MinUpdateEvery,
		ModuleRegistry:     module.DefaultRegistry,
		Out:                os.Stdout,
	}
}

type readyOnce struct {
	c    chan struct{}
	once sync.Once
}

func newReadyOnce() *readyOnce {
	return &readyOnce{c: make(chan struct{})}
}

func (c *readyOnce) ready()                 { c.once.Do(func() { close(c.c) }) }
func (c *readyOnce) isReady() chan struct{} { return c.c }

// Run
func (p *Plugin) Run() {
	go signalHandling()
	go keepAlive()
	run(p)
}

func run(p *Plugin) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	reload(ch, p)
}

func reload(ch chan os.Signal, p *Plugin) {
	ctx, cancel := context.WithCancel(context.Background())
	once := newReadyOnce()
	go p.run(ctx, once)

	<-ch
	cancel()
	<-once.isReady()
	reload(ch, p)
}

func (p *Plugin) run(ctx context.Context, once *readyOnce) {
	defer once.ready()
	discoverer, builder, saver, runner, err := p.setup()
	if err != nil {
		return
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
}

func (p *Plugin) disable() {
	_ = netdataapi.New(p.Out).DISABLE()
}

func signalHandling() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)

	sig := <-ch
	log.Infof("received %s signal (%d). Terminating...", sig, sig)

	switch sig {
	case syscall.SIGPIPE:
		os.Exit(1)
	default:
		os.Exit(0)
	}
}

func keepAlive() {
	if isTerminal {
		return
	}
	for range time.Tick(time.Second) {
		_, _ = fmt.Fprint(os.Stdout, "\n")
	}
}
