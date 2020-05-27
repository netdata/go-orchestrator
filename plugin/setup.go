package plugin

import (
	"io"
	"os"

	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/job/discovery"
	"github.com/netdata/go-orchestrator/job/discovery/dummy"
	"github.com/netdata/go-orchestrator/job/discovery/file"
	"github.com/netdata/go-orchestrator/module"

	"gopkg.in/yaml.v2"
)

func defaultConfig() config {
	return config{
		Enabled:    true,
		DefaultRun: true,
		MaxProcs:   0,
		Modules:    nil,
	}
}

type config struct {
	Enabled    bool            `yaml:"enabled"`
	DefaultRun bool            `yaml:"default_run"`
	MaxProcs   int             `yaml:"max_procs"`
	Modules    map[string]bool `yaml:"modules"`
}

func (p *Plugin) loadPluginConfig() config {
	p.Info("loading config")

	if len(p.ConfDir) == 0 {
		p.Info("config dir provided, will use defaults")
		return defaultConfig()
	}

	name := p.Name + ".conf"
	p.Infof("looking for '%s' in %v", name, p.ConfDir)

	path, err := p.ConfDir.Find(name)
	if err != nil || path == "" {
		p.Warning("couldn't find config, will use defaults")
		return defaultConfig()
	}
	p.Infof("found '%s", path)

	cfg := defaultConfig()
	if err := loadYAML(&cfg, path); err != nil {
		p.Warningf("couldn't load config '%s': %v, will use defaults", path, err)
		return defaultConfig()
	}
	p.Info("config successfully loaded")
	return cfg
}

func (p *Plugin) loadEnabledModules(cfg config) module.Registry {
	all := p.RunModule == "all" || p.RunModule == ""
	enabled := module.Registry{}

	for name, creator := range p.ModuleRegistry {
		if !all && p.RunModule != name {
			continue
		}
		if all && creator.Disabled && !cfg.isExplicitlyEnabled(name) {
			p.Infof("module '%s' disabled by default, should be explicitly enabled in the config", name)
			continue
		}
		if all && !cfg.isImplicitlyEnabled(name) {
			p.Infof("module '%s' disabled in the config file", name)
			continue
		}
		enabled[name] = creator
	}
	return enabled
}

func (p *Plugin) buildDiscoveryConf(enabled module.Registry) discovery.Config {
	reg := confgroup.Registry{}
	for name, creator := range enabled {
		reg.Register(name, confgroup.Default{
			MinUpdateEvery:     p.MinUpdateEvery,
			UpdateEvery:        creator.UpdateEvery,
			AutoDetectionRetry: creator.AutoDetectionRetry,
			Priority:           creator.Priority,
		})
	}

	var readPaths, dummyPaths []string

	if len(p.ModulesConfDir) == 0 {
		p.Info("modules conf dir not provided, will use default config for enabled modules")
		for name := range enabled {
			dummyPaths = append(dummyPaths, name)
		}
		return discovery.Config{
			Registry: reg,
			Dummy:    dummy.Config{Names: dummyPaths}}
	}

	for modName := range enabled {
		name := modName + ".conf"
		p.Infof("looking for '%s' in %v", name, p.ModulesConfDir)

		path, err := p.ModulesConfDir.Find(name + ".conf")
		if err != nil {
			p.Infof("couldn't find '%s' config, will use default config", modName)
			dummyPaths = append(dummyPaths, modName)
		} else {
			p.Infof("found '%s", path)
			readPaths = append(readPaths, path)
		}
	}

	return discovery.Config{
		Registry: reg,
		File: file.Config{
			Read:  readPaths,
			Watch: p.ModulesSDConfPath,
		},
		Dummy: dummy.Config{
			Names: dummyPaths,
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
	type plain config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}

	for key, value := range m {
		switch key {
		case "enabled", "default_run", "max_procs", "modules":
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
