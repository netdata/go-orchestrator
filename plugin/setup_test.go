package plugin

import (
	"runtime"
	"testing"

	"github.com/netdata/go-plugin/cli"
	"github.com/netdata/go-plugin/module"
	"github.com/netdata/go-plugin/pkg/multipath"

	"github.com/stretchr/testify/assert"
)

func TestPlugin_SetupNoName(t *testing.T) {
	p := New()
	assert.False(t, p.Setup())
}

func TestPlugin_SetupNoOptions(t *testing.T) {
	p := New()
	p.Name = "test"
	assert.False(t, p.Setup())
}

func TestPlugin_SetupNoConfigPath(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{}
	p.ConfigPath = nil
	assert.False(t, p.Setup())

	p.ConfigPath = multipath.New()
	assert.False(t, p.Setup())
}

func TestPlugin_SetupNoRegistry(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{}
	p.ConfigPath = multipath.New("./tests")
	assert.False(t, p.Setup())

	p.Registry = make(module.Registry)
	assert.False(t, p.Setup())
}

func TestPlugin_SetupNoConfig(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{"module1": module.Creator{}}
	assert.False(t, p.Setup())
}

func TestPlugin_SetupBrokenConfig(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{"module1": module.Creator{}}
	p.configName = "go.d.conf-broken.yml"
	assert.False(t, p.Setup())
}

func TestPlugin_SetupEmptyConfig(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{Module: "all"}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{"module1": module.Creator{}}
	p.configName = "go.d.conf-empty.yml"

	assert.True(t, p.Setup())
}

func TestPlugin_SetupDisabledInConfig(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{"module1": module.Creator{}}
	p.configName = "go.d.conf-disabled.yml"
	assert.False(t, p.Setup())
}

func TestPlugin_SetupNoModulesToRun(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{Module: "module3"}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{"module1": module.Creator{}}
	p.configName = "go.d.conf.yml"

	assert.Len(t, p.modules, 0)
	assert.False(t, p.Setup())
}

func TestPlugin_SetupSetGOMAXPROCS(t *testing.T) {
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{Module: "all"}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{"module1": module.Creator{}, "module2": module.Creator{}}
	p.Config.MaxProcs = 1
	p.configName = "go.d.conf.yml"
	assert.True(t, p.Setup())
	assert.Equal(t, p.Config.MaxProcs, runtime.GOMAXPROCS(0))
}

func TestPlugin_Setup(t *testing.T) {
	// OK all
	p := New()
	p.Name = "test"
	p.Option = &cli.Option{Module: "all"}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{
		"module1": module.Creator{},
		"module2": module.Creator{},
		"module3": module.Creator{}}
	p.configName = "go.d.conf.yml"
	assert.True(t, p.Setup())
	assert.Len(t, p.modules, 3)

	// OK all with disabled by default
	p = New()
	p.Name = "test"
	p.Option = &cli.Option{Module: "all"}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{
		"module1": module.Creator{},
		"module2": module.Creator{},
		"module3": module.Creator{DisabledByDefault: true},
	}
	p.configName = "go.d.conf.yml"
	assert.True(t, p.Setup())
	assert.Len(t, p.modules, 2)

	// OK specific
	p = New()
	p.Name = "test"
	p.Option = &cli.Option{Module: "module2"}
	p.ConfigPath = multipath.New("./tests")
	p.Registry = module.Registry{"module1": module.Creator{}, "module2": module.Creator{}}
	p.configName = "go.d.conf.yml"
	assert.True(t, p.Setup())
	assert.Len(t, p.modules, 1)

}
