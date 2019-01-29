package plugin

import (
	"fmt"
	"runtime"

	"github.com/netdata/go-plugin/pkg/multipath"
)

func (p *Plugin) Setup() bool {
	if p.Name == "" {
		log.Critical("name not set")
		return false
	}
	if p.Option == nil {
		log.Critical("cli options not set")
		return false
	}
	if p.Option.ConfigDir != "" {
		p.ConfigPath = multipath.New(p.Option.ConfigDir)
	}
	if len(p.ConfigPath) == 0 {
		log.Critical("config path not set or empty")
		return false
	}
	if len(p.Registry) == 0 {
		log.Critical("registry not set or empty")
		return false
	}

	if p.configName == "" {
		p.configName = p.Name + ".conf"
	}
	configFile, err := p.ConfigPath.Find(p.configName)
	if err != nil {
		log.Critical("find config file error: ", err)
		return false
	}

	if err := loadYAML(p.Config, configFile); err != nil {
		log.Critical("loadYAML config error: ", err)
		return false
	}

	if !p.Config.Enabled {
		log.Info("disabled in configuration file")
		_, _ = fmt.Fprintln(p.Out, "DISABLE")
		return false
	}

	isAll := p.Option.Module == "all"
	for name, creator := range p.Registry {
		if !isAll && p.Option.Module != name {
			continue
		}
		if isAll && creator.DisabledByDefault && !p.Config.isModuleEnabled(name, true) {
			log.Infof("module '%s' disabled by default", name)
			continue
		}
		if isAll && !p.Config.isModuleEnabled(name, false) {
			log.Infof("module '%s' disabled in configuration file", name)
			continue
		}
		p.modules[name] = creator
	}

	if len(p.modules) == 0 {
		log.Critical("no modules to run")
		return false
	}

	if p.Config.MaxProcs > 0 {
		log.Infof("maximum number of used CPUs set to %d", p.Config.MaxProcs)
		runtime.GOMAXPROCS(p.Config.MaxProcs)
	} else {
		log.Infof("maximum number of used CPUs %d", runtime.NumCPU())
	}

	log.Infof("minimum update every %d", p.Option.UpdateEvery)

	return true
}
