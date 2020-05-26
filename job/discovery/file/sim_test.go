package file

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type (
	discoverySim struct {
		discovery      *Discovery
		beforeRun      func()
		afterRun       []simRunAction
		expectedGroups []*confgroup.Group
	}
	simRunAction struct {
		name   string
		delay  time.Duration
		action func()
	}
)

func (sim discoverySim) run(t *testing.T) {
	t.Helper()
	require.NotNil(t, sim.discovery)

	if sim.beforeRun != nil {
		sim.beforeRun()
	}

	in, out := make(chan []*confgroup.Group), make(chan []*confgroup.Group)
	go sim.collectGroups(t, in, out)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	go sim.discovery.Run(ctx, in)
	time.Sleep(time.Millisecond * 200)

	for _, after := range sim.afterRun {
		if after.action != nil {
			after.action()
			time.Sleep(after.delay)
		}
	}
	groups := <-out

	assert.Equal(t, sim.expectedGroups, groups)
}

func (sim discoverySim) collectGroups(t *testing.T, in, out chan []*confgroup.Group) {
	timeout := time.Second * 10
	var groups []*confgroup.Group
loop:
	for {
		select {
		case inGroups := <-in:
			if groups = append(groups, inGroups...); len(groups) >= len(sim.expectedGroups) {
				break loop
			}
		case <-time.After(timeout):
			t.Logf("discovery %s timed out after %s, got %d groups, expected %d, some events are skipped",
				sim.discovery.discoverers, timeout, len(groups), len(sim.expectedGroups))
			break loop
		}
	}
	out <- groups
}

type tmpDir struct {
	dir string
	t   *testing.T
}

func newTmpDir(t *testing.T, pattern string) *tmpDir {
	dir, err := ioutil.TempDir(os.TempDir(), pattern)
	require.NoError(t, err)
	return &tmpDir{dir: dir, t: t}
}

func (d *tmpDir) cleanup() {
	assert.NoError(d.t, os.RemoveAll(d.dir))
}

func (d *tmpDir) createFile(pattern string) string {
	f, err := ioutil.TempFile(d.dir, pattern)
	require.NoError(d.t, err)
	f.Close()
	return f.Name()
}

func (d *tmpDir) join(filename string) string {
	return filepath.Join(d.dir, filename)
}

func (d *tmpDir) removeFile(filename string) {
	err := os.Remove(filename)
	require.NoError(d.t, err)
}

func (d *tmpDir) writeYAML(filename string, in interface{}) {
	bs, err := yaml.Marshal(in)
	require.NoError(d.t, err)
	err = ioutil.WriteFile(filename, bs, 0644)
	require.NoError(d.t, err)
}
