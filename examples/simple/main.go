package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/netdata/go-orchestrator/cli"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/logger"
	"github.com/netdata/go-orchestrator/plugin"
)

var version = "v0.0.1-example"

type example struct{ module.Base }

func (example) Cleanup() {}

func (example) Init() bool { return true }

func (example) Check() bool { return true }

func (example) Charts() *module.Charts {
	return &module.Charts{
		{
			ID:    "random",
			Title: "A Random Number", Units: "random", Fam: "random",
			Dims: module.Dims{
				{ID: "random0", Name: "random 0"},
				{ID: "random1", Name: "random 1"},
			},
		},
	}
}

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
	if opt.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	module.Register("example", module.Creator{Create: func() module.Module { return &example{} }})

	p, err := plugin.New(plugin.Config{
		Name:           "test.d",
		MinUpdateEvery: 1,
		UseModule:      "",
		//ConfDir: []string{
		//	"/Users/ilyam/Projects/golang/go-orchestrator/examples/simple/",
		//},
		SDConfPath: []string{
			"/opt/sd.yml",
			//"/Users/ilyam/Projects/golang/go-orchestrator/examples/simple/sd.yml",
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	p.Run()
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
