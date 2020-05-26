package file

import (
	"testing"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/module"

	"github.com/stretchr/testify/assert"
)

func TestWatcher_String(t *testing.T) {
	assert.NotEmpty(t, NewWatcher(confgroup.Registry{}, nil))
}

func TestNewWatcher(t *testing.T) {
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
		t.Run(name, func(t *testing.T) { assert.NotNil(t, NewWatcher(test.reg, test.paths)) })
	}
}

func TestWatcher_Run(t *testing.T) {
	tests := map[string]func(tmp *tmpDir) discoverySim{
		"file exists before start": func(tmp *tmpDir) discoverySim {
			reg := confgroup.Registry{
				"module": {},
			}
			cfg := sdConfig{
				{
					"name":   "name",
					"module": "module",
				},
			}
			filename := tmp.join("module.conf")
			discovery := prepareDiscovery(t, Config{
				Registry: reg,
				Watch:    []string{tmp.join("*.conf")},
			})
			expected := []*confgroup.Group{
				{
					Source: filename,
					Configs: []confgroup.Config{
						{
							"name":                "name",
							"module":              "module",
							"update_every":        module.UpdateEvery,
							"autodetection_retry": module.AutoDetectionRetry,
							"priority":            module.Priority,
						},
					},
				},
			}

			sim := discoverySim{
				discovery: discovery,
				beforeRun: func() {
					tmp.writeYAML(filename, cfg)
				},
				expectedGroups: expected,
			}
			return sim
		},
		"add file": func(tmp *tmpDir) discoverySim {
			reg := confgroup.Registry{
				"module": {},
			}
			cfg := sdConfig{
				{
					"name":   "name",
					"module": "module",
				},
			}
			filename := tmp.join("module.conf")
			discovery := prepareDiscovery(t, Config{
				Registry: reg,
				Watch:    []string{tmp.join("*.conf")},
			})
			expected := []*confgroup.Group{
				{
					Source: filename,
					Configs: []confgroup.Config{
						{
							"name":                "name",
							"module":              "module",
							"update_every":        module.UpdateEvery,
							"autodetection_retry": module.AutoDetectionRetry,
							"priority":            module.Priority,
						},
					},
				},
			}

			sim := discoverySim{
				discovery: discovery,
				afterRun: []simRunAction{
					{
						name:  "add file",
						delay: time.Millisecond * 100,
						action: func() {
							tmp.writeYAML(filename, cfg)
						},
					},
				},
				expectedGroups: expected,
			}
			return sim
		},
		"remove file": func(tmp *tmpDir) discoverySim {
			reg := confgroup.Registry{
				"module": {},
			}
			cfg := sdConfig{
				{
					"name":   "name",
					"module": "module",
				},
			}
			filename := tmp.join("module.conf")
			discovery := prepareDiscovery(t, Config{
				Registry: reg,
				Watch:    []string{tmp.join("*.conf")},
			})
			expected := []*confgroup.Group{
				{
					Source: filename,
					Configs: []confgroup.Config{
						{
							"name":                "name",
							"module":              "module",
							"update_every":        module.UpdateEvery,
							"autodetection_retry": module.AutoDetectionRetry,
							"priority":            module.Priority,
						},
					},
				},
				{
					Source:  filename,
					Configs: nil,
				},
			}

			sim := discoverySim{
				discovery: discovery,
				beforeRun: func() {
					tmp.writeYAML(filename, cfg)
				},
				afterRun: []simRunAction{
					{
						name:  "remove file",
						delay: time.Millisecond * 100,
						action: func() {
							tmp.removeFile(filename)
						},
					},
				},
				expectedGroups: expected,
			}
			return sim
		},
		"change file": func(tmp *tmpDir) discoverySim {
			reg := confgroup.Registry{
				"module": {},
			}
			cfgOrig := sdConfig{
				{
					"name":   "name",
					"module": "module",
				},
			}
			cfgChanged := sdConfig{
				{
					"name":   "name_changed",
					"module": "module",
				},
			}
			filename := tmp.join("module.conf")
			discovery := prepareDiscovery(t, Config{
				Registry: reg,
				Watch:    []string{tmp.join("*.conf")},
			})
			expected := []*confgroup.Group{
				{
					Source: filename,
					Configs: []confgroup.Config{
						{
							"name":                "name",
							"module":              "module",
							"update_every":        module.UpdateEvery,
							"autodetection_retry": module.AutoDetectionRetry,
							"priority":            module.Priority,
						},
					},
				},
				{
					Source: filename,
					Configs: []confgroup.Config{
						{
							"name":                "name_changed",
							"module":              "module",
							"update_every":        module.UpdateEvery,
							"autodetection_retry": module.AutoDetectionRetry,
							"priority":            module.Priority,
						},
					},
				},
			}

			sim := discoverySim{
				discovery: discovery,
				beforeRun: func() {
					tmp.writeYAML(filename, cfgOrig)
				},
				afterRun: []simRunAction{
					{
						name:  "overwrite file",
						delay: time.Millisecond * 100,
						action: func() {
							tmp.writeYAML(filename, cfgChanged)
						},
					},
				},
				expectedGroups: expected,
			}
			return sim
		},
	}

	for name, createSim := range tests {
		t.Run(name, func(t *testing.T) {
			tmp := newTmpDir(t, "watch-run-*")
			defer tmp.cleanup()

			createSim(tmp).run(t)
		})
	}
}
