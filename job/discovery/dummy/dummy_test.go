package dummy

import (
	"testing"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscovery(t *testing.T) {
	tests := map[string]struct {
		cfg     Config
		wantErr bool
	}{
		"valid config": {
			cfg: Config{
				Registry: confgroup.Registry{"module1": confgroup.Default{}},
				Names:    []string{"module1", "module2"},
			},
		},
		"invalid config, registry not set": {
			cfg: Config{
				Names: []string{"module1", "module2"},
			},
			wantErr: true,
		},
		"invalid config, names not set": {
			cfg: Config{
				Names: []string{"module1", "module2"},
			},
			wantErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			d, err := NewDiscovery(test.cfg)

			if test.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, d)
			}
		})
	}
}

func TestDiscovery_Run(t *testing.T) {
	tests := map[string]struct {
		discovery      Discovery
		expectedGroups []*confgroup.Group
	}{
		"": {},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_ = test
		})
	}
}
