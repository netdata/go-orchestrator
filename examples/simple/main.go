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
	metrics := make(map[string]int64)
	metrics["random0"] = rand.Int63n(100)
	metrics["random1"] = rand.Int63n(100)

	return metrics
}

func main() {
	opt := parseCLI()

	if opt.Debug {
		logger.SetSeverity(logger.DEBUG)
	}

	p := createPlugin(opt)

	if !p.Setup() {
		return
	}

	p.Serve()
}

func createPlugin(opt *cli.Option) *orchestrator.Orchestrator {
	p := orchestrator.New()
	p.Name = "go.d"
	p.Option = opt
	p.Registry = make(module.Registry)
	p.Registry.Register("example", module.Creator{Create: func() module.Module { return &example{} }})

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
