package build

import (
	"testing"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/stretchr/testify/assert"
)

// TODO: tech debt
func TestJobCache_put(t *testing.T) {
	tests := map[string][]struct {
		name          string
		groups        []confgroup.Group
		wantAdd       []confgroup.Config
		wantRemove    []confgroup.Config
		wantGlobalLen int // TODO: ?
		wantSourceLen int // TODO: ?
	}{
		//"add configs from 1 source": {
		//	{
		//		groups: []confgroup.Group{
		//			prepareGroup("source1",
		//				prepareCfg("name1", "module1"),
		//				prepareCfg("name2", "module1"),
		//				prepareCfg("name3", "module1"),
		//			),
		//		},
		//		wantAdd: []confgroup.Config{
		//			prepareCfg("name1", "module1"),
		//			prepareCfg("name2", "module1"),
		//			prepareCfg("name3", "module1"),
		//		},
		//		wantRemove:    nil,
		//		wantGlobalLen: 1,
		//		wantSourceLen: 1,
		//	},
		//},
		"add config from 1 source them remove them partially": {
			{
				groups: []confgroup.Group{
					prepareGroup("source1",
						prepareCfg("name1", "module1"),
						prepareCfg("name2", "module1"),
						prepareCfg("name3", "module1"),
					),
					prepareGroup("source1",
						prepareCfg("name3", "module1"),
					),
				},
				wantAdd: []confgroup.Config{
					prepareCfg("name1", "module1"),
					prepareCfg("name2", "module1"),
					prepareCfg("name3", "module1"),
				},
				wantRemove: []confgroup.Config{
					prepareCfg("name1", "module1"),
					prepareCfg("name2", "module1"),
				},
				wantGlobalLen: 1,
				wantSourceLen: 1,
			},
		},
		//"2 groups with different configs": {
		//	{
		//		groups: []confgroup.Group{
		//			prepareGroup("source1",
		//				prepareCfg("name1", "module1"),
		//			),
		//			prepareGroup("source2",
		//				prepareCfg("name2", "module2"),
		//			),
		//		},
		//		wantAdd: []confgroup.Config{
		//			prepareCfg("name1", "module1"),
		//			prepareCfg("name2", "module2"),
		//		},
		//		wantRemove:    nil,
		//		wantGlobalLen: 1,
		//		wantSourceLen: 2,
		//	},
		//},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cache := newGroupCache()

			for i, step := range test {
				var added, removed []confgroup.Config

				for _, group := range step.groups {
					a, r := cache.put(&group)
					added = append(added, a...)
					removed = append(removed, r...)
				}

				assert.Equalf(t, step.wantAdd, added, "added configs, step '%s' %d", step.name, i+1)
				assert.Equalf(t, step.wantRemove, removed, "removed configs, step '%s' %d", step.name, i+1)
				//assert.Lenf(t, cache.global, step.wantGlobalLen, "global cache, step %d", i+1)
				//assert.Lenf(t, cache.source, step.wantSourceLen, "per source cache, step %d", i+1)
			}
		})
	}
}

func prepareGroup(source string, cfgs ...confgroup.Config) confgroup.Group {
	return confgroup.Group{
		Configs: cfgs,
		Source:  source,
	}
}

func prepareCfg(name, module string) confgroup.Config {
	return confgroup.Config{
		"name":   name,
		"module": module,
	}
}
