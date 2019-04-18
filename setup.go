package orchestrator

import (
	"fmt"
	"os"
	"runtime"

	"github.com/netdata/go-orchestrator/logger"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"github.com/mattn/go-isatty"
)

func (o *Orchestrator) Setup() bool {
	if o.Name == "" {
		log.Critical("name not set")
		return false
	}

	//TODO: fix
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logger.SetPluginName(o.Name, log)
	}

	if o.Option == nil {
		log.Critical("cli options not set")
		return false
	}
	if len(o.Option.ConfigDir) != 0 {
		o.ConfigPath = multipath.New(o.Option.ConfigDir...)
	}
	if len(o.ConfigPath) == 0 {
		log.Critical("config path not set or empty")
		return false
	}
	if len(o.Registry) == 0 {
		log.Critical("registry not set or empty")
		return false
	}

	if o.configName == "" {
		o.configName = o.Name + ".conf"
	}

	configFile, err := o.ConfigPath.Find(o.configName)

	if err != nil {
		if !multipath.IsNotFound(err) {
			log.Criticalf("find config file error : %v", err)
			return false
		}
		log.Warningf("find config file error : %v, will use default configuration", err)
	} else {
		if err := loadYAML(o.Config, configFile); err != nil {
			log.Criticalf("loadYAML config error : %v", err)
			return false
		}
	}

	if !o.Config.Enabled {
		log.Info("disabled in configuration file")
		_, _ = fmt.Fprintln(o.Out, "DISABLE")
		return false
	}

	isAll := o.Option.Module == "all"
	for name, creator := range o.Registry {
		if !isAll && o.Option.Module != name {
			continue
		}
		if isAll && creator.Disabled && !o.Config.isModuleEnabled(name, true) {
			log.Infof("module '%s' disabled by default", name)
			continue
		}
		if isAll && !o.Config.isModuleEnabled(name, false) {
			log.Infof("module '%s' disabled in configuration file", name)
			continue
		}
		o.modules[name] = creator
	}

	if len(o.modules) == 0 {
		log.Critical("no modules to run")
		return false
	}

	if o.Config.MaxProcs > 0 {
		log.Infof("maximum number of used CPUs set to %d", o.Config.MaxProcs)
		runtime.GOMAXPROCS(o.Config.MaxProcs)
	} else {
		log.Infof("maximum number of used CPUs %d", runtime.NumCPU())
	}

	log.Infof("minimum update every %d", o.Option.UpdateEvery)

	return true
}
