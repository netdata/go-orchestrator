package main

import (
	"flag"
	"math/rand"
	"os"

	"github.com/netdata/go-orchestrator"
	"github.com/netdata/go-orchestrator/cli"
	"github.com/netdata/go-orchestrator/logger"
	"github.com/netdata/go-orchestrator/module"
)

var charts = module.Charts{
	{
		ID:    "random",
		Title: "A Random Number", Units: "random", Fam: "random",
		Dims: module.Dims{
			{ID: "random0", Name: "random 0"},
			{ID: "random1", Name: "random 1"},
		},
	},
}

type example struct{ module.Base }

func (example) Cleanup() {}

func (example) Init() bool { return true }

func (example) Check() bool { return true }

func (example) Charts() *module.Charts { return charts.Copy() }

func (e *example) Collect() map[string]int64 {
	return map[string]int64{
		"random0": rand.Int63n(100),
		"random1": rand.Int63n(100),
	}
}

func main() {
	opt := parseCLI()

	if opt.Debug {
		logger.SetSeverity(logger.DEBUG)
	}
	
	module.Register("example", module.Creator{Create: func() module.Module { return &example{} }})

	p := newPlugin(opt)

	if !p.Setup() {
		return
	}

	p.Serve()
}

func newPlugin(opt *cli.Option) *orchestrator.Orchestrator {
	p := orchestrator.New()
	p.Name = "test.d"
	p.Option = opt

	return p
}

func parseCLI() *cli.Option {
	opt, err := cli.Parse(os.Args)
	if err != nil {
		if err != flag.ErrHelp {
			os.Exit(1)
		}
		os.Exit(0)
	}

	return opt
}
