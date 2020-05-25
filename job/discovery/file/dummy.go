package file

import (
	"context"

	"github.com/netdata/go-orchestrator/job/confgroup"
)

type Dummy struct {
	reg   confgroup.Registry
	names []string
}

func NewDummy(req confgroup.Registry, names []string) *Dummy {
	return &Dummy{
		reg:   req,
		names: names,
	}
}

func (d Dummy) Run(ctx context.Context, in chan<- []*confgroup.Group) {
	select {
	case <-ctx.Done():
	case in <- d.groups():
	}
	close(in)
}

func (d Dummy) groups() (groups []*confgroup.Group) {
	for _, name := range d.names {
		groups = append(groups, d.newCfgGroup(name))
	}
	return groups
}

func (d Dummy) newCfgGroup(name string) *confgroup.Group {
	def, ok := d.reg.Lookup(name)
	if !ok {
		return nil
	}

	cfg := confgroup.Config{}
	cfg.Set("module", name)
	cfg.Apply(def)

	group := &confgroup.Group{
		Configs: []confgroup.Config{cfg},
		Source:  name,
	}
	return group
}
