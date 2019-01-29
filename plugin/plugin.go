package plugin

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/netdata/go-plugin/cli"
	"github.com/netdata/go-plugin/logger"
	"github.com/netdata/go-plugin/module"
	"github.com/netdata/go-plugin/pkg/multipath"
	"github.com/netdata/go-plugin/plugin/ticker"

	"github.com/mattn/go-isatty"
)

var (
	log = logger.New("plugin", "main")

	cd, _             = os.Getwd()
	defaultConfigPath = multipath.New(
		os.Getenv("NETDATA_USER_CONFIG_DIR"),
		os.Getenv("NETDATA_STOCK_CONFIG_DIR"),
		path.Join(cd, "/../../../../etc/netdata"),            // if installed in /opt
		path.Join(cd, "/../../../../usr/lib/netdata/conf.d"), // if installed in /opt
	)
)

// Job is an interface that represents a job.
type Job interface {
	FullName() string
	ModuleName() string
	Name() string

	AutoDetectionRetry() int

	Panicked() bool

	Init() bool
	Check() bool
	PostCheck() bool

	Tick(clock int)

	Start()
	Stop()
}

type Config struct {
	Enabled    bool            `yaml:"enabled"`
	DefaultRun bool            `yaml:"default_run"`
	MaxProcs   int             `yaml:"max_procs"`
	Modules    map[string]bool `yaml:"modules"`
}

func (c Config) isModuleEnabled(module string, explicit bool) bool {
	if run, ok := c.Modules[module]; ok {
		return run
	}
	if explicit {
		return false
	}
	return c.DefaultRun
}

// New creates Plugin.
func New() *Plugin {
	return &Plugin{
		ConfigPath:       multipath.New(defaultConfigPath...),
		Config:           &Config{Enabled: true, DefaultRun: true},
		Registry:         module.DefaultRegistry,
		Out:              os.Stdout,
		modules:          make(module.Registry),
		jobStartCh:       make(chan Job),
		jobStartLoopStop: make(chan struct{}),
		mainLoopStop:     make(chan struct{}),
	}
}

// Plugin represents plugin.
type Plugin struct {
	Name       string
	Out        io.Writer
	Registry   module.Registry
	Option     *cli.Option
	ConfigPath multipath.MultiPath
	Config     *Config

	configName string

	jobStartCh       chan Job
	jobStartLoopStop chan struct{}
	mainLoopStop     chan struct{}

	modules   module.Registry
	loopQueue loopQueue
}

// RemoveFromQueue removes job from the loop queue by full name.
func (p *Plugin) RemoveFromQueue(fullName string) {
	if job := p.loopQueue.remove(fullName); job != nil {
		job.Stop()
	}
}

// Serve Serve
func (p *Plugin) Serve() {
	go shutdownTask()

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		go heartbeatTask()
	}

	go p.jobStartLoop()

	for _, job := range p.createJobs() {
		p.jobStartCh <- job
	}

	p.mainLoop()
}

func (p *Plugin) mainLoop() {
	log.Info("start main loop")
	tk := ticker.New(time.Second)

LOOP:
	for {
		select {
		case <-p.mainLoopStop:
			break LOOP
		case clock := <-tk.C:
			p.runOnce(clock)
		}
	}
}

func (p *Plugin) runOnce(clock int) {
	log.Debugf("tick %d", clock)
	p.loopQueue.notify(clock)
}

func (p *Plugin) stop() {
	p.jobStartLoopStop <- struct{}{}
	p.mainLoopStop <- struct{}{}
}

func shutdownTask() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGPIPE)

	switch <-signalChan {
	case syscall.SIGINT:
		log.Info("SIGINT received. Terminating...")
	case syscall.SIGHUP:
		log.Info("SIGHUP received. Terminating...")
	case syscall.SIGPIPE:
		log.Critical("SIGPIPE received. Terminating...")
		os.Exit(1)
	}
	os.Exit(0)
}

func heartbeatTask() {
	t := time.Tick(time.Second)
	for range t {
		_, _ = fmt.Fprint(os.Stdout, "\n")
	}
}
