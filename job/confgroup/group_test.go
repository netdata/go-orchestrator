package confgroup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Name(t *testing.T) {
	tests := map[string]struct {
		cfg      Config
		expected interface{}
	}{
		"string": {
			cfg:      Config{"name": "name"},
			expected: "name",
		},
		"empty string": {
			cfg:      Config{"name": ""},
			expected: "",
		},
		"not string": {
			cfg:      Config{"name": 0},
			expected: "",
		},
		"not set": {
			cfg:      Config{},
			expected: "",
		},
		"nil cfg": {
			expected: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.cfg.Name())
		})
	}
}

func TestConfig_Module(t *testing.T) {
	tests := map[string]struct {
		cfg      Config
		expected interface{}
	}{
		"string": {
			cfg:      Config{"module": "module"},
			expected: "module",
		},
		"empty string": {
			cfg:      Config{"module": ""},
			expected: "",
		},
		"not string": {
			cfg:      Config{"module": 0},
			expected: "",
		},
		"not set": {
			cfg:      Config{},
			expected: "",
		},
		"nil cfg": {
			expected: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.cfg.Module())
		})
	}
}

func TestConfig_FullName(t *testing.T) {
	tests := map[string]struct {
		cfg      Config
		expected interface{}
	}{
		"name == module": {
			cfg:      Config{"name": "name", "module": "name"},
			expected: "name",
		},
		"name != module": {
			cfg:      Config{"name": "name", "module": "module"},
			expected: "module_name",
		},
		"nil cfg": {
			expected: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.cfg.FullName())
		})
	}
}

func TestConfig_UpdateEvery(t *testing.T) {
	tests := map[string]struct {
		cfg      Config
		expected interface{}
	}{
		"int": {
			cfg:      Config{"update_every": 1},
			expected: 1,
		},
		"not int": {
			cfg:      Config{"update_every": "1"},
			expected: 0,
		},
		"not set": {
			cfg:      Config{},
			expected: 0,
		},
		"nil cfg": {
			expected: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.cfg.UpdateEvery())
		})
	}
}

func TestConfig_AutoDetectionRetry(t *testing.T) {
	tests := map[string]struct {
		cfg      Config
		expected interface{}
	}{
		"int": {
			cfg:      Config{"autodetection_retry": 1},
			expected: 1,
		},
		"not int": {
			cfg:      Config{"autodetection_retry": "1"},
			expected: 0,
		},
		"not set": {
			cfg:      Config{},
			expected: 0,
		},
		"nil cfg": {
			expected: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.cfg.AutoDetectionRetry())
		})
	}
}

func TestConfig_Priority(t *testing.T) {
	tests := map[string]struct {
		cfg      Config
		expected interface{}
	}{
		"int": {
			cfg:      Config{"priority": 1},
			expected: 1,
		},
		"not int": {
			cfg:      Config{"priority": "1"},
			expected: 0,
		},
		"not set": {
			cfg:      Config{},
			expected: 0,
		},
		"nil cfg": {
			expected: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.cfg.Priority())
		})
	}
}

func TestConfig_Hash(t *testing.T) {
	cfg := Config{"name": "name", "module": "module"}
	assert.NotZero(t, cfg.Hash())
}

func TestConfig_Set(t *testing.T) {
	expected := Config{"name": "name"}

	actual := Config{}
	actual.Set("name", "name")

	assert.Equal(t, expected, actual)
}

func TestConfig_Apply(t *testing.T) {
	tests := map[string]struct {
		def         Default
		origCfg     Config
		expectedCfg Config
	}{}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.origCfg.Apply(test.def)

			assert.Equal(t, test.expectedCfg, test.origCfg)
		})
	}
}
