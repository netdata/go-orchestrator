package plugin

import (
	"io"
	"os"

	"github.com/netdata/go-orchestrator/job/build"
	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/job/discovery"
	"github.com/netdata/go-orchestrator/job/discovery/file"
	"github.com/netdata/go-orchestrator/job/run"
	"github.com/netdata/go-orchestrator/job/state"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"gopkg.in/yaml.v2"
)

type config struct {
	Enabled    bool            `yaml:"enabled"`
	DefaultRun bool            `yaml:"default_run"`
	MaxProcs   int             `yaml:"max_procs"`
	Modules    map[string]bool `yaml:"modules"`
}

func (p *Plugin) setup() (*discovery.Manager, *build.Manager, *state.Manager, *run.Manager, error) {
	cfg := p.loadPluginConfig()
	enabled := p.loadEnabledModules(cfg)
	discCfg := p.buildDiscoveryConf(enabled)

	dm, err := discovery.NewManager(discCfg)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rm := run.NewManager()
	bm := build.NewManager()
	bm.Runner = rm
	bm.Modules = enabled
	bm.Out = p.Out
	bm.PluginName = p.Name

	var sm *state.Manager
	if !isTerminal && p.StateFile != "" {
		sm = state.NewManager(p.StateFile)
		bm.Saver = sm
		if st, err := state.Load(p.StateFile); err == nil {
			bm.PrevState = st
		}
	}
	return dm, bm, sm, rm, nil
}

func (p *Plugin) loadPluginConfig() config {
	defaultConf := config{
		Enabled:    true,
		DefaultRun: true,
		MaxProcs:   0,
		Modules:    nil,
	}
	if len(p.ConfPath) == 0 {
		log.Info("plugin config path not provided, will use defaults")
		return defaultConf
	}

	name := p.Name + ".conf"
	log.Infof("looking for '%s' in %s", name, p.ConfPath)

	path, err := p.ConfPath.Find(name)
	if err != nil || path == "" {
		log.Warning("couldn't find plugin config, will use defaults")
		return defaultConf
	}
	log.Infof("found '%s", path)

	if err := loadYAML(defaultConf, path); err != nil {
		log.Warningf("couldn't load '%s': %v, will use defaults", path, err)
	}
	return defaultConf
}

func (p *Plugin) loadEnabledModules(cfg config) module.Registry {
	all := p.RunModule == "all" || p.RunModule == ""
	enabled := module.Registry{}

	for name, creator := range p.ModuleRegistry {
		if !all && p.RunModule != name {
			continue
		}
		if all && creator.Disabled && !cfg.isExplicitlyEnabled(name) {
			log.Infof("module '%s' disabled by default", name)
			continue
		}
		if all && !cfg.isImplicitlyEnabled(name) {
			log.Infof("module '%s' disabled in configuration file", name)
			continue
		}
		enabled[name] = creator
	}
	return enabled
}

func (p *Plugin) buildDiscoveryConf(enabled module.Registry) discovery.Config {
	var paths, dummyPaths []string
	reg := confgroup.Registry{}

	for name, creator := range enabled {
		reg.Register(name, confgroup.Default{
			MinUpdateEvery:     p.MinUpdateEvery,
			UpdateEvery:        creator.UpdateEvery,
			AutoDetectionRetry: creator.AutoDetectionRetry,
			Priority:           creator.Priority,
		})
	}

	if len(p.ModulesConfPath) == 0 {
		for name := range enabled {
			dummyPaths = append(dummyPaths, name)
		}
	} else {
		for name := range enabled {
			path, err := p.ModulesConfPath.Find(name + ".conf")
			if err != nil && !multipath.IsNotFound(err) {
				continue
			}

			if err != nil {
				dummyPaths = append(dummyPaths, name)
			} else {
				paths = append(paths, path)
			}
		}
	}
	return discovery.Config{
		Registry: reg,
		File: file.Config{
			Dummy: dummyPaths,
			Read:  paths,
			Watch: p.ModulesSDConfFiles,
		},
	}
}

func (c config) isExplicitlyEnabled(moduleName string) bool {
	return c.isEnabled(moduleName, true)
}

func (c config) isImplicitlyEnabled(moduleName string) bool {
	return c.isEnabled(moduleName, false)
}

func (c config) isEnabled(moduleName string, explicit bool) bool {
	if enabled, ok := c.Modules[moduleName]; ok {
		return enabled
	}
	if explicit {
		return false
	}
	return c.DefaultRun
}

func (c *config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type cfg config
	cc := cfg(*c)
	if err := unmarshal(&cc); err != nil {
		return err
	}
	*c = config(cc)

	var m map[string]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}

	for key, value := range m {
		switch key {
		case "enabled", "default_run", "max_procs", "enabledModules":
			continue
		}
		var b bool
		if in, err := yaml.Marshal(value); err != nil || yaml.Unmarshal(in, &b) != nil {
			continue
		}
		if c.Modules == nil {
			c.Modules = make(map[string]bool)
		}
		c.Modules[key] = b
	}
	return nil
}

func loadYAML(conf interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = yaml.NewDecoder(f).Decode(conf); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return nil
}
