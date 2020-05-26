package file

import (
	"testing"

	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/module"

	"github.com/stretchr/testify/assert"
)

func TestNewReader(t *testing.T) {
	tests := map[string]struct {
		reg   confgroup.Registry
		paths []string
	}{
		"empty inputs": {
			reg:   confgroup.Registry{},
			paths: []string{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) { assert.NotNil(t, NewReader(test.reg, test.paths)) })
	}
}

func TestReader_Run(t *testing.T) {
	tmp := newTmpDir(t, "reader-run-*")
	defer tmp.cleanup()

	module1 := tmp.join("module1.conf")
	module2 := tmp.join("module2.conf")
	tmp.writeYAML(module1, staticConfig{
		Jobs: []confgroup.Config{{"name": "name"}},
	})
	tmp.writeYAML(module2, staticConfig{
		Jobs: []confgroup.Config{{"name": "name"}},
	})
	reg := confgroup.Registry{
		"module1": {},
		"module2": {},
	}
	discovery := prepareDiscovery(t, Config{
		Registry: reg,
		Read:     []string{module1, module2},
	})
	expected := []*confgroup.Group{
		{
			Source: module1,
			Configs: []confgroup.Config{
				{
					"name":                "name",
					"module":              "module1",
					"update_every":        module.UpdateEvery,
					"autodetection_retry": module.AutoDetectionRetry,
					"priority":            module.Priority,
				},
			},
		},
		{
			Source: module2,
			Configs: []confgroup.Config{
				{
					"name":                "name",
					"module":              "module2",
					"update_every":        module.UpdateEvery,
					"autodetection_retry": module.AutoDetectionRetry,
					"priority":            module.Priority,
				},
			},
		},
	}

	sim := discoverySim{
		discovery:      discovery,
		expectedGroups: expected,
	}

	sim.run(t)
}
