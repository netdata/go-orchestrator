package file

import (
	"testing"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	//tests := map[string]func(*tmpDir) discoverySim{
	//	"": func(td *tmpDir) discoverySim {
	//		return discoverySim{}
	//	},
	//}
	//
	//for name, createSim := range tests {
	//	t.Run(name, func(t *testing.T) {
	//		td := newTmpDir(t, "netdata-god-discovery-file-read-*")
	//		defer td.cleanup()
	//		createSim(td).run(t)
	//	})
	//}
}

func prepareDiscovery(t *testing.T, cfg Config) *Discovery {
	d, err := NewDiscovery(cfg)
	require.NoError(t, err)
	return d
}
