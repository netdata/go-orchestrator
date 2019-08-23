package orchestrator

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"gopkg.in/yaml.v2"
)

const (
	defaultUpdateEvery        = 1
	defaultAutoDetectionRetry = 0
	DefaultJobPriority        = 70000
)

func newModuleConfig() *moduleConfig {
	return &moduleConfig{
		UpdateEvery:        defaultUpdateEvery,
		AutoDetectionRetry: defaultAutoDetectionRetry,
		Priority:           DefaultJobPriority,
	}
}

type moduleConfig struct {
	UpdateEvery        int                      `yaml:"update_every"`
	AutoDetectionRetry int                      `yaml:"autodetection_retry"`
	Priority           int                      `yaml:"priority"`
	Jobs               []map[string]interface{} `yaml:"jobs"`

	name string
}

func (m *moduleConfig) setGlobalDefaults(defaults module.Defaults) {
	if defaults.UpdateEvery > 0 {
		m.UpdateEvery = defaults.UpdateEvery
	}
	if defaults.AutoDetectionRetry > 0 {
		m.AutoDetectionRetry = defaults.AutoDetectionRetry
	}
	if defaults.Priority > 0 {
		m.Priority = defaults.Priority
	}
}

func (m *moduleConfig) updateJobs(minUpdateEvery int) {
	for _, job := range m.Jobs {
		if _, ok := job["update_every"]; !ok {
			job["update_every"] = m.UpdateEvery
		}

		if _, ok := job["autodetection_retry"]; !ok {
			job["autodetection_retry"] = m.AutoDetectionRetry
		}

		if _, ok := job["priority"]; !ok {
			job["priority"] = m.Priority
		}

		if v, ok := job["update_every"].(int); ok && v < minUpdateEvery {
			job["update_every"] = minUpdateEvery
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
	if creator, ok := o.Registry[name]; ok {
		modConf.setGlobalDefaults(creator.Defaults)
	}
	modConf.name = name

	configPath, err := o.ConfigPath.Find(fmt.Sprintf("%s/%s.conf", dirName, name))

	if err != nil {
		if !multipath.IsNotFound(err) {
			log.Errorf("skipping '%s': %v", name, err)
			return nil
		}

		log.Warningf("'%s': %v, will use default 1 job configuration", name, err)
		modConf.Jobs = []map[string]interface{}{{}}
		return modConf
	}

	if err = loadYAML(modConf, configPath); err != nil {
		log.Errorf("skipping '%s': %v", name, err)
		return nil
	}

	if len(modConf.Jobs) == 0 {
		log.Errorf("skipping '%s': config 'jobs' section is empty or not exist", name)
		return nil
	}

	return modConf
}

func (o *Orchestrator) createModuleJobs(modConf *moduleConfig, js *jobsStatuses) []Job {
	var jobs []Job

	creator := o.Registry[modConf.name]
	modConf.updateJobs(o.Option.UpdateEvery)

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

		job := module.NewJob(o.Name, modConf.name, mod, o.Out)

		if err := unmarshal(conf, job); err != nil {
			log.Errorf("skipping %s[%s]: %s", modConf.name, jobName(conf), err)
			continue
		}

		if js != nil && js.contains(Job(job)) && job.AutoDetectEvery == 0 {
			log.Infof("%s[%s] was active on  previous run, applying recovering settings", job.ModuleName(), job.Name())
			job.AutoDetectTries = 11
			job.AutoDetectEvery = 30
		}

		jobs = append(jobs, job)
	}

	return jobs
}

func (o *Orchestrator) createJobs() []Job {
	var jobs []Job

	js, err := o.loadJobsStatuses()
	if err != nil {
		log.Warning(err)
	}

	for name := range o.modules {
		conf := o.loadModuleConfig(name)
		if conf == nil {
			continue
		}

		for _, job := range o.createModuleJobs(conf, js) {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

func (o *Orchestrator) loadJobsStatuses() (*jobsStatuses, error) {
	if o.varLibDir == "" {
		return nil, nil
	}

	name := path.Join(o.varLibDir, jobStatusesFile)
	v, err := loadJobsStatusesFromFile(name)
	if err != nil {
		return nil, fmt.Errorf("error on loading '%s' : %v", name, err)
	}
	return v, nil
}

func unmarshal(conf interface{}, module interface{}) error {
	b, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
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
