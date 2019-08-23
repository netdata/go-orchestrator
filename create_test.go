package orchestrator

import (
	"fmt"
	"path"
	"testing"

	"github.com/netdata/go-orchestrator/cli"
	"github.com/netdata/go-orchestrator/module"
	"github.com/netdata/go-orchestrator/pkg/multipath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadModuleConfigNoConfig(t *testing.T) {
	o := New()
	o.Name = "test.d"
	o.ConfigPath = multipath.New("./testdata")
	assert.NotNil(t, o.loadModuleConfig("no_config"))
}

func Test_loadModuleConfigBrokenConfig(t *testing.T) {
	o := New()
	o.Name = "test.d"
	o.ConfigPath = multipath.New("./testdata")
	assert.Nil(t, o.loadModuleConfig("module-broken"))
}

func Test_loadModuleConfigNoJobs(t *testing.T) {
	o := New()
	o.Name = "test.d"
	o.ConfigPath = multipath.New("./testdata")
	assert.Nil(t, o.loadModuleConfig("module-no-jobs"))
}

func Test_loadModuleConfig(t *testing.T) {
	o := New()
	o.Name = "test.d"
	o.ConfigPath = multipath.New("./testdata")
	o.ModulesConfigDirName = "test.d"
	conf := o.loadModuleConfig("module1")
	require.NotNil(t, conf)
	assert.Equal(t, 3, len(conf.Jobs))
}

func Test_loadModuleConfigNotFound(t *testing.T) {
	o := New()
	o.Name = "test.d"
	o.ConfigPath = multipath.New("./testdata")
	o.ModulesConfigDirName = "test_not_exist.d"
	conf := o.loadModuleConfig("module1")
	require.NotNil(t, conf)
	assert.Equal(t, 1, len(conf.Jobs))
}

func Test_createModuleJobs(t *testing.T) {
	o := New()
	o.Name = "test.d"
	o.ConfigPath = multipath.New("./testdata")
	o.Option = &cli.Option{}
	reg := make(module.Registry)
	reg.Register(
		"module1",
		module.Creator{Create: func() module.Module { return &module.MockModule{} }},
	)

	o.Registry = reg
	conf := newModuleConfig()
	conf.Jobs = []map[string]interface{}{{}, {}, {}}
	conf.name = "module1"
	assert.Len(t, o.createModuleJobs(conf, nil), 3)
}

func Test_createModuleJobsWithJobsStatuses(t *testing.T) {
	o := New()
	o.Name = "test.d"
	o.ConfigPath = multipath.New("./testdata")
	o.Option = &cli.Option{}
	reg := make(module.Registry)
	reg.Register(
		"module1",
		module.Creator{Create: func() module.Module { return &module.MockModule{} }},
	)

	o.Registry = reg
	conf := newModuleConfig()
	conf.Jobs = []map[string]interface{}{{}, {}, {}}
	conf.name = "module1"

	js, err := loadJobsStatusesFromFile(path.Join("./testdata", "god-jobs-statuses.json"))
	require.NoError(t, err)
	fmt.Println(js.items)

	jobs := o.createModuleJobs(conf, js)
	assert.Len(t, jobs, 3)
	for _, job := range jobs {
		j := job.(*module.Job)
		assert.Equal(t, 11, j.AutoDetectTries)
		assert.Equal(t, 30, j.AutoDetectEvery)
	}
}

func TestPluginConfig_isModuleEnabled(t *testing.T) {
	modName1 := "modName1"
	modName2 := "modName2"
	modName3 := "modName3"

	conf := Config{
		DefaultRun: true,
		Modules: map[string]bool{
			modName1: true,
			modName2: false,
		},
	}

	assert.True(t, conf.isModuleEnabled(modName1, false))
	assert.False(t, conf.isModuleEnabled(modName2, false))
	assert.Equal(
		t,
		conf.DefaultRun,
		conf.isModuleEnabled(modName3, false),
	)

	assert.True(t, conf.isModuleEnabled(modName1, true))
	assert.False(t, conf.isModuleEnabled(modName2, true))
	assert.Equal(
		t,
		!conf.DefaultRun,
		conf.isModuleEnabled(modName3, true),
	)

	conf.DefaultRun = false

	assert.True(t, conf.isModuleEnabled(modName1, false))
	assert.False(t, conf.isModuleEnabled(modName2, false))
	assert.Equal(
		t,
		conf.DefaultRun,
		conf.isModuleEnabled(modName3, false),
	)

	assert.True(t, conf.isModuleEnabled(modName1, true))
	assert.False(t, conf.isModuleEnabled(modName2, true))
	assert.Equal(
		t,
		conf.DefaultRun,
		conf.isModuleEnabled(modName3, true),
	)

}

func TestModuleConfig_updateJobs(t *testing.T) {
	conf := newModuleConfig()
	conf.Jobs = []map[string]interface{}{
		{"name": "job1"},
		{"name": "job2", "update_every": 10},
	}
	conf.updateJobs(0)

	assert.Equal(
		t,
		[]map[string]interface{}{
			{
				"name":                "job1",
				"update_every":        defaultUpdateEvery,
				"autodetection_retry": defaultAutoDetectionRetry,
				"priority":            DefaultJobPriority,
			},
			{
				"name":                "job2",
				"update_every":        10,
				"autodetection_retry": defaultAutoDetectionRetry,
				"priority":            DefaultJobPriority,
			},
		},
		conf.Jobs,
	)
}

func TestModuleConfig_UpdateJobsRewriteModuleUpdateEvery(t *testing.T) {
	conf := newModuleConfig()
	conf.setGlobalDefaults(module.Defaults{UpdateEvery: 20})
	conf.Jobs = []map[string]interface{}{
		{"name": "job1"},
		{"name": "job2", "update_every": 10},
	}
	conf.updateJobs(0)

	assert.Equal(
		t,
		[]map[string]interface{}{
			{
				"name":                "job1",
				"update_every":        20,
				"autodetection_retry": defaultAutoDetectionRetry,
				"priority":            DefaultJobPriority,
			},
			{
				"name":                "job2",
				"update_every":        10,
				"autodetection_retry": defaultAutoDetectionRetry,
				"priority":            DefaultJobPriority,
			},
		},
		conf.Jobs,
	)
}

func TestModuleConfig_UpdateJobsRewritePluginUpdateEvery(t *testing.T) {
	conf := newModuleConfig()
	conf.Jobs = []map[string]interface{}{
		{"name": "job1"},
		{"name": "job2", "update_every": 10},
	}
	conf.updateJobs(5)

	assert.Equal(
		t,
		[]map[string]interface{}{
			{
				"name":                "job1",
				"update_every":        5,
				"autodetection_retry": defaultAutoDetectionRetry,
				"priority":            DefaultJobPriority,
			},
			{
				"name":                "job2",
				"update_every":        10,
				"autodetection_retry": defaultAutoDetectionRetry,
				"priority":            DefaultJobPriority,
			},
		},
		conf.Jobs,
	)
}
