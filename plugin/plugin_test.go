package plugin

import (
	"testing"
	"time"

	"github.com/netdata/go-plugin/cli"
	"github.com/netdata/go-plugin/module"
	"github.com/netdata/go-plugin/pkg/multipath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	p := New()
	assert.IsType(t, (*Plugin)(nil), p)
	assert.NotNil(t, p.Out)
	assert.NotNil(t, p.ConfigPath)
	assert.NotNil(t, p.Registry)
	assert.NotNil(t, p.Config)
}

func TestPlugin_Serve(t *testing.T) {
	p := New()
	p.Name = "test"

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

	p.Option = &cli.Option{Module: "all"}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{
		"module1": module.Creator{Create: func() module.Module { return mod() }},
		"module2": module.Creator{Create: func() module.Module { return mod() }},
	}
	p.configName = "go.d.conf.yml"

	require.True(t, p.Setup())

	go p.Serve()

	time.Sleep(time.Second * 3)

	p.stop()

	for _, job := range p.loopQueue.queue {
		job.Stop()
	}
}
