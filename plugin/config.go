package plugin

import "gopkg.in/yaml.v2"

type config struct {
	Enabled    bool            `yaml:"enabled"`
	DefaultRun bool            `yaml:"default_run"`
	MaxProcs   int             `yaml:"max_procs"`
	Modules    map[string]bool `yaml:"enabledModules"`
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

func (c config) isModuleExplicitlyEnabled(name string) bool {
	return c.isModuleEnabled(name, true)
}

func (c config) isModuleImplicitlyEnabled(name string) bool {
	return c.isModuleEnabled(name, false)
}

func (c config) isModuleEnabled(name string, explicit bool) bool {
	if run, ok := c.Modules[name]; ok {
		return run
	}
	if explicit {
		return false
	}
	return c.DefaultRun
}
