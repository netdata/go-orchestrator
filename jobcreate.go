package orchestrator

import (
	"fmt"
	"io"
	"os"

	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"gopkg.in/yaml.v2"
)

func newModuleConfig() *moduleConfig {
	return &moduleConfig{
		UpdateEvery:        1,
		AutoDetectionRetry: 0,
	}
}

type moduleConfig struct {
	UpdateEvery        int                      `yaml:"update_every"`
	AutoDetectionRetry int                      `yaml:"autodetection_retry"`
	Jobs               []map[string]interface{} `yaml:"jobs"`

	name string
}

func (m *moduleConfig) updateJobs(moduleUpdateEvery, pluginUpdateEvery int) {
	if moduleUpdateEvery > 0 {
		m.UpdateEvery = moduleUpdateEvery
	}

	for _, job := range m.Jobs {
		if _, ok := job["update_every"]; !ok {
			job["update_every"] = m.UpdateEvery
		}

		if _, ok := job["autodetection_retry"]; !ok {
			job["autodetection_retry"] = m.AutoDetectionRetry
		}

		if v, ok := job["update_every"].(int); ok && v < pluginUpdateEvery {
			job["update_every"] = pluginUpdateEvery
		}
	}
}

func (o *Orchestrator) loadModuleConfig(name string) *moduleConfig {
	log.Infof("loading '%s' configuration", name)

	dirName := o.ModulesConfigDirName
	if dirName == "" {
		dirName = o.Name
	}

	modConf := newModuleConfig()
	modConf.name = name

	configPath, err := o.ConfigPath.Find(fmt.Sprintf("%s/%s.conf", dirName, name))

	if err != nil {
		if !multipath.IsNotFound(err) {
			log.Warningf("skipping '%s': %v", name, err)
			return nil
		}

		log.Warningf("'%s': %v, will use default 1 job configuration", name, err)
		modConf.Jobs = []map[string]interface{}{{}}
		return modConf
	}

	if err = loadYAML(modConf, configPath); err != nil {
		log.Warningf("skipping '%s': %v", name, err)
		return nil
	}

	if len(modConf.Jobs) == 0 {
		log.Warningf("skipping '%s': config 'jobs' section is empty or not exist", name)
		return nil
	}

	return modConf
}

func (o *Orchestrator) createModuleJobs(modConf *moduleConfig) []Job {
	var jobs []Job

	creator := o.Registry[modConf.name]
	modConf.updateJobs(creator.UpdateEvery, o.Option.UpdateEvery)

	jobName := func(conf map[string]interface{}) interface{} {
		if name, ok := conf["name"]; ok {
			return name
		}
		return "unnamed"
	}

	for _, conf := range modConf.Jobs {
		mod := creator.Create()

		if err := unmarshal(conf, mod); err != nil {
			log.Errorf("skipping %s[%s]: %s", modConf.name, jobName(conf), err)
			continue
		}

		job := module.NewJob(o.Name, modConf.name, mod, o.Out, o)

		if err := unmarshal(conf, job); err != nil {
			log.Errorf("skipping %s[%s]: %s", modConf.name, jobName(conf), err)
			continue
		}

		jobs = append(jobs, job)
	}

	return jobs
}

func (o *Orchestrator) createJobs() []Job {
	var jobs []Job

	for name := range o.modules {
		conf := o.loadModuleConfig(name)
		if conf == nil {
			continue
		}

		for _, job := range o.createModuleJobs(conf) {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

func unmarshal(conf interface{}, module interface{}) error {
	b, _ := yaml.Marshal(conf)
	return yaml.Unmarshal(b, module)
}

func loadYAML(conf interface{}, filename string) error {
	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		log.Debug("open file ", filename, ": ", err)
		return err
	}

	if err = yaml.NewDecoder(file).Decode(conf); err != nil {
		if err == io.EOF {
			log.Debug("config file is empty")
			return nil
		}
		log.Debug("read YAML ", filename, ": ", err)
		return err
	}

	return nil
}
