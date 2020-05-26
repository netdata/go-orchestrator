package file

import (
	"testing"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/stretchr/testify/assert"
)

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
		"": func(tmp *tmpDir) discoverySim {

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
