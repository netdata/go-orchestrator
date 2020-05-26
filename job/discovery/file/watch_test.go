package file

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
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

// TODO: tech dept
func TestWatcher_Run(t *testing.T) {
	tests := map[string]func(t2 *testing.T){
		"": func(t2 *testing.T) {
			tempDir := os.TempDir()
			workDir, err := ioutil.TempDir(tempDir, "netdata-discovery-watch-test-*")
			require.NoError(t, err)

			defer func() {
				if err := os.Remove(workDir); err != nil {
					t.Logf("couldnt delete work dir '%s': %v", workDir, err)
				}
			}()

			f, err := ioutil.TempFile(workDir, "sd-*")
			require.NoError(t, err)
			defer f.Close()
			defer os.Remove(f.Name())

			bs, err := yaml.Marshal(sdConfig{
				{"name": "name1"},
				{"name": "name2"},
			})
			require.NoError(t, err)

			err = ioutil.WriteFile(f.Name(), bs, 0644)
			require.NoError(t, err)

			bs, err = ioutil.ReadFile(f.Name())
			fmt.Println(string(bs))

			fmt.Println("-----------------------")

			bs, err = yaml.Marshal(sdConfig{
				{"name": "name3"},
				{"name": "name4"},
			})
			require.NoError(t, err)

			err = ioutil.WriteFile(f.Name(), bs, 0644)
			require.NoError(t, err)

			bs, err = ioutil.ReadFile(f.Name())
			fmt.Println(string(bs))
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) { test(t) })
	}
}
