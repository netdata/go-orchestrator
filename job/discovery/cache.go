package discovery

import (
	"github.com/netdata/go-orchestrator/job/confgroup"
)

type cache map[string]*confgroup.Group

func newCache() *cache {
	return &cache{}
}

func (c cache) update(groups []*confgroup.Group) {
	for _, group := range groups {
		if group != nil {
			c[group.Source] = group
		}
	}
}

func (c cache) reset() {
	for key := range c {
		delete(c, key)
	}
}

func (c cache) groups() []*confgroup.Group {
	groups := make([]*confgroup.Group, 0, len(c))
	for _, group := range c {
		groups = append(groups, group)
	}
	return groups
}
