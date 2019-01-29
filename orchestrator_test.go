package orchestrator

import (
	"sync"
	"testing"
	"time"

	"github.com/netdata/go-orchestrator/cli"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	o := New()
	assert.IsType(t, (*Orchestrator)(nil), o)
	assert.NotNil(t, o.Out)
	assert.NotNil(t, o.ConfigPath)
	assert.NotNil(t, o.Registry)
	assert.NotNil(t, o.Config)
}

func TestOrchestrator_Serve(t *testing.T) {
	o := New()
	o.Name = "go.d"

	mod := func() module.Module {
		return &module.MockModule{
			InitFunc:  func() bool { return true },
			CheckFunc: func() bool { return true },
			ChartsFunc: func() *module.Charts {
				return &module.Charts{
					&module.Chart{
						ID:    "id",
						Title: "title",
						Units: "units",
						Dims: module.Dims{
							{ID: "id1"},
							{ID: "id2"},
						},
					},
				}
			},
			CollectFunc: func() map[string]int64 {
				return map[string]int64{
					"id1": 1,
					"id2": 2,
				}
			},
		}
	}

	o.Option = &cli.Option{Module: "all"}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{
		"module1": module.Creator{Create: func() module.Module { return mod() }},
		"module2": module.Creator{Create: func() module.Module { return mod() }},
	}
	o.configName = "go.d.conf.yml"

	require.True(t, o.Setup())

	go o.Serve()

	time.Sleep(time.Second * 3)

	o.stop()

	for _, job := range o.loopQueue.queue {
		job.Stop()
	}
}

func TestLoopQueue_add(t *testing.T) {
	var l loopQueue
	var wg sync.WaitGroup

	workers := 10
	addNum := 1000

	f := func() {
		for i := 0; i < addNum; i++ {
			l.add(nil)
		}
		wg.Done()
	}

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go f()
	}

	wg.Wait()

	assert.Equal(t, workers*addNum, len(l.queue))
}
