package file

import (
	"testing"

	"github.com/netdata/go-orchestrator/job/confgroup"
	"github.com/netdata/go-orchestrator/module"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	const (
		jobDef = 11
		cfgDef = 22
		modDef = 33
	)
	tests := map[string]func(t *testing.T, td *tmpDir){
		"static, default: +job +conf +module": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"module": {
					UpdateEvery:        modDef,
					AutoDetectionRetry: modDef,
					Priority:           modDef,
				},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name":                "name",
				"update_every":        jobDef,
				"autodetection_retry": jobDef,
				"priority":            jobDef,
			}
			content := staticConfig{
				Default: confgroup.Default{
					UpdateEvery:        cfgDef,
					AutoDetectionRetry: cfgDef,
					Priority:           cfgDef,
				},
				Jobs: []confgroup.Config{cfg},
			}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"name":                "name",
						"module":              "module",
						"update_every":        jobDef,
						"autodetection_retry": jobDef,
						"priority":            jobDef,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"static, default: +job +conf +module (merge all)": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"module": {
					Priority: modDef,
				},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name":         "name",
				"update_every": jobDef,
			}
			content := staticConfig{
				Default: confgroup.Default{
					AutoDetectionRetry: cfgDef,
				},
				Jobs: []confgroup.Config{cfg},
			}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"name":                "name",
						"module":              "module",
						"update_every":        jobDef,
						"autodetection_retry": cfgDef,
						"priority":            modDef,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"static, default: -job +conf +module": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"module": {
					UpdateEvery:        modDef,
					AutoDetectionRetry: modDef,
					Priority:           modDef,
				},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name": "name",
			}
			content := staticConfig{
				Default: confgroup.Default{
					UpdateEvery:        cfgDef,
					AutoDetectionRetry: cfgDef,
					Priority:           cfgDef,
				},
				Jobs: []confgroup.Config{cfg},
			}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"name":                "name",
						"module":              "module",
						"update_every":        cfgDef,
						"autodetection_retry": cfgDef,
						"priority":            cfgDef,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"static, default: -job -conf +module": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"module": {
					UpdateEvery:        modDef,
					AutoDetectionRetry: modDef,
					Priority:           modDef,
				},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name": "name",
			}
			content := staticConfig{
				Jobs: []confgroup.Config{cfg},
			}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"name":                "name",
						"module":              "module",
						"autodetection_retry": modDef,
						"priority":            modDef,
						"update_every":        modDef,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"static, default: -job -conf -module (+global)": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"module": {},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name": "name",
			}
			content := staticConfig{
				Jobs: []confgroup.Config{cfg},
			}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"name":                "name",
						"module":              "module",
						"autodetection_retry": module.AutoDetectionRetry,
						"priority":            module.Priority,
						"update_every":        module.UpdateEvery,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"sd, default: +job +module": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"sd_module": {
					UpdateEvery:        modDef,
					AutoDetectionRetry: modDef,
					Priority:           modDef,
				},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name":                "name",
				"module":              "sd_module",
				"update_every":        jobDef,
				"autodetection_retry": jobDef,
				"priority":            jobDef,
			}
			content := sdConfig{cfg}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"module":              "sd_module",
						"name":                "name",
						"update_every":        jobDef,
						"autodetection_retry": jobDef,
						"priority":            jobDef,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"sd, default: -job +module": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"sd_module": {
					UpdateEvery:        modDef,
					AutoDetectionRetry: modDef,
					Priority:           modDef,
				},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name":   "name",
				"module": "sd_module",
			}
			content := sdConfig{cfg}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"name":                "name",
						"module":              "sd_module",
						"update_every":        modDef,
						"autodetection_retry": modDef,
						"priority":            modDef,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"sd, default: -job -module (+global)": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"sd_module": {},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name":   "name",
				"module": "sd_module",
			}
			content := sdConfig{cfg}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source: filename,
				Configs: []confgroup.Config{
					{
						"name":                "name",
						"module":              "sd_module",
						"update_every":        module.UpdateEvery,
						"autodetection_retry": module.AutoDetectionRetry,
						"priority":            module.Priority,
					},
				},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"sd, job has no 'module' or 'module' is empty": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"sd_module": {},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name": "name",
			}
			content := sdConfig{cfg}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source:  filename,
				Configs: []confgroup.Config{},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"conf registry has no module": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"sd_module": {},
			}

			filename := td.join("module.conf")
			cfg := confgroup.Config{
				"name":   "name",
				"module": "module",
			}
			content := sdConfig{cfg}
			td.writeYAML(filename, content)

			expected := &confgroup.Group{
				Source:  filename,
				Configs: []confgroup.Config{},
			}

			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Equal(t, expected, groups)
		},
		"empty file": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{
				"module": {},
			}

			filename := td.createFile("empty-*")
			groups, err := parseFile(reg, filename)

			require.NoError(t, err)
			assert.Nil(t, groups)
		},
		"unknown format": func(t *testing.T, td *tmpDir) {
			reg := confgroup.Registry{}

			filename := td.createFile("unknown-format-*")
			td.writeYAML(filename, "unknown")
			_, err := parseFile(reg, filename)

			assert.Error(t, err)
		},
	}

	for name, scenario := range tests {
		t.Run(name, func(t *testing.T) {
			td := newTmpDir(t, "netdata-god-discovery-file-parseFile-*")
			defer td.cleanup()
			scenario(t, td)
		})
	}
}
