package orchestrator

import (
	"runtime"
	"testing"

	"github.com/netdata/go-orchestrator/cli"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"github.com/stretchr/testify/assert"
)

func TestOrchestrator_SetupNoName(t *testing.T) {
	o := New()
	assert.False(t, o.Setup())
}

func TestOrchestrator_SetupNoOptions(t *testing.T) {
	o := New()
	o.Name = "test"
	assert.False(t, o.Setup())
}

func TestOrchestrator_SetupNoConfigPath(t *testing.T) {
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{}
	o.ConfigPath = nil
	assert.False(t, o.Setup())

	o.ConfigPath = multipath.New()
	assert.False(t, o.Setup())
}

func TestOrchestrator_SetupNoRegistry(t *testing.T) {
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{}
	o.ConfigPath = multipath.New("./testdata")
	assert.False(t, o.Setup())

	o.Registry = make(module.Registry)
	assert.False(t, o.Setup())
}

func TestOrchestrator_SetupNoConfig(t *testing.T) {
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{"module1": module.Creator{}}
	assert.False(t, o.Setup())
}

func TestPlugin_SetupBrokenConfig(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{}
	p.ConfigPath = multipath.New("./testdata")
	p.Registry = module.Registry{"module1": module.Creator{}}
	p.configName = "go.d.conf-broken.yml"
	assert.False(t, p.Setup())
}

func TestOrchestrator_SetupEmptyConfig(t *testing.T) {
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{Module: "all"}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{"module1": module.Creator{}}
	o.configName = "go.d.conf-empty.yml"

	assert.True(t, o.Setup())
}

func TestOrchestrator_SetupDisabledInConfig(t *testing.T) {
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{"module1": module.Creator{}}
	o.configName = "go.d.conf-disabled.yml"
	assert.False(t, o.Setup())
}

func TestOrchestrator_SetupNoModulesToRun(t *testing.T) {
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{Module: "module3"}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{"module1": module.Creator{}}
	o.configName = "go.d.conf.yml"

	assert.Len(t, o.modules, 0)
	assert.False(t, o.Setup())
}

func TestOrchestrator_SetupSetGOMAXPROCS(t *testing.T) {
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{Module: "all"}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{"module1": module.Creator{}, "module2": module.Creator{}}
	o.Config.MaxProcs = 1
	o.configName = "go.d.conf.yml"
	assert.True(t, o.Setup())
	assert.Equal(t, o.Config.MaxProcs, runtime.GOMAXPROCS(0))
}

func TestOrchestrator_Setup(t *testing.T) {
	// OK all
	o := New()
	o.Name = "test"
	o.Option = &cli.Option{Module: "all"}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{
		"module1": module.Creator{},
		"module2": module.Creator{},
		"module3": module.Creator{}}
	o.configName = "go.d.conf.yml"
	assert.True(t, o.Setup())
	assert.Len(t, o.modules, 3)

	// OK all with disabled by default
	o = New()
	o.Name = "test"
	o.Option = &cli.Option{Module: "all"}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{
		"module1": module.Creator{},
		"module2": module.Creator{},
		"module3": module.Creator{DisabledByDefault: true},
	}
	o.configName = "go.d.conf.yml"
	assert.True(t, o.Setup())
	assert.Len(t, o.modules, 2)

	// OK specific
	o = New()
	o.Name = "test"
	o.Option = &cli.Option{Module: "module2"}
	o.ConfigPath = multipath.New("./testdata")
	o.Registry = module.Registry{"module1": module.Creator{}, "module2": module.Creator{}}
	o.configName = "go.d.conf.yml"
	assert.True(t, o.Setup())
	assert.Len(t, o.modules, 1)

}
