package orchestrator

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/netdata/go-orchestrator/cli"
	"github.com/netdata/go-orchestrator/logger"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"github.com/mattn/go-isatty"
)

var (
	log             = logger.New("plugin", "main", "main")
	jobStatusesFile = "god-jobs-statuses.json"
)

// Job is an interface that represents a job.
type Job interface {
	FullName() string
	ModuleName() string
	Name() string
	AutoDetection() bool
	AutoDetectionEvery() int
	RetryAutoDetection() bool
	Panicked() bool
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

// New creates Orchestrator.
func New() *Orchestrator {
	return &Orchestrator{
		ConfigPath: multipath.New(
			os.Getenv("NETDATA_USER_CONFIG_DIR"),
			os.Getenv("NETDATA_STOCK_CONFIG_DIR"),
		),
		Config:       &Config{Enabled: true, DefaultRun: true},
		Registry:     module.DefaultRegistry,
		Out:          os.Stdout,
		varLibDir:    os.Getenv("NETDATA_LIB_DIR"),
		modules:      make(module.Registry),
		jobStartCh:   make(chan Job),
		jobStartStop: make(chan struct{}),
		mainLoopStop: make(chan struct{}),
		jobsStatuses: &jobsStatuses{mux: new(sync.Mutex), items: make(map[string]map[string]string)},
	}
}

// Orchestrator represents orchestrator.
type Orchestrator struct {
	Name                 string
	Out                  io.Writer
	Registry             module.Registry
	Option               *cli.Option
	ConfigPath           multipath.MultiPath
	Config               *Config
	ModulesConfigDirName string

	varLibDir  string
	configName string

	jobStartCh   chan Job
	jobStartStop chan struct{}
	mainLoopStop chan struct{}

	jobStatusWriter io.Writer
	jobsStatuses    *jobsStatuses
	modules         module.Registry
	loopQueue       loopQueue
}

// Serve Serve
func (o *Orchestrator) Serve() {
	go signalHandling()

	isAtty := isatty.IsTerminal(os.Stdout.Fd())

	if !isAtty {
		go keepAlive()
	}

	go o.jobStartLoop()

	for _, job := range o.createJobs() {
		o.jobStartCh <- job
	}

	if !isAtty && o.varLibDir != "" {
		w := &fileWriter{path: path.Join(o.varLibDir, jobStatusesFile)}
		s := newJobsStatusesSaver(w, o.jobsStatuses, time.Second*10)
		go s.mainLoop()
		defer s.stop()
	}

	o.mainLoop()
}

func (o *Orchestrator) mainLoop() {
	log.Info("start main loop")
	tk := NewTicker(time.Second)

LOOP:
	for {
		select {
		case <-o.mainLoopStop:
			break LOOP
		case clock := <-tk.C:
			o.runOnce(clock)
		}
	}
}

func (o *Orchestrator) runOnce(clock int) {
	log.Debugf("tick %d", clock)
	o.loopQueue.notify(clock)
}

func (o *Orchestrator) stop() {
	o.jobStartStop <- struct{}{}
	o.mainLoopStop <- struct{}{}
}

func signalHandling() {
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

func keepAlive() {
	t := time.Tick(time.Second)
	for range t {
		_, _ = fmt.Fprint(os.Stdout, "\n")
	}
}

type loopQueue struct {
	mux   sync.Mutex
	queue []Job
}

func (q *loopQueue) add(job Job) {
	q.mux.Lock()
	defer q.mux.Unlock()

	q.queue = append(q.queue, job)
}

func (q *loopQueue) notify(clock int) {
	q.mux.Lock()
	defer q.mux.Unlock()

	for _, job := range q.queue {
		job.Tick(clock)
	}
}
