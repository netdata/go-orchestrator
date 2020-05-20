package build

import (
	"testing"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/stretchr/testify/assert"
)

// TODO: tech debt
func TestJobCache_put(t *testing.T) {
	tests := map[string][]struct {
		groups        []confgroup.Group
		wantAdd       []confgroup.Config
		wantRemove    []confgroup.Config
		wantGlobalLen int
		wantSourceLen int
	}{
		"qwe": {
			{
				groups: []confgroup.Group{
					{
						Configs: []confgroup.Config{
							prepareCfg("name1", "module1"),
							prepareCfg("name1", "module1"),
							prepareCfg("name1", "module1"),
						},
						Source: "source1",
					},
				},
				wantAdd: []confgroup.Config{
					prepareCfg("name1", "module1"),
				},
				wantRemove:    nil,
				wantGlobalLen: 1,
				wantSourceLen: 1,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cache := newGroupCache()
			var added, removed []confgroup.Config

			for i, step := range test {
				added, removed = added[:0], removed[:0]

				for _, group := range step.groups {
					a, r := cache.put(&group)
					added = append(added, a...)
					removed = append(removed, r...)
				}

				assert.Equalf(t, step.wantAdd, added, "added configurations, step %d", i+1)
				assert.Equalf(t, step.wantRemove, removed, "removed configurations, step %d", i+1)
				assert.Lenf(t, cache.global, step.wantGlobalLen, "global cache, step %d", i+1)
				assert.Lenf(t, cache.source, step.wantSourceLen, "per source cache, step %d", i+1)
			}
		})
	}
}

func prepareCfg(name, module string) confgroup.Config {
	return confgroup.Config{
		"name":   name,
		"module": module,
	}
}
