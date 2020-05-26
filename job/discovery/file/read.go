package file

import (
	"context"
	"os"
	"path/filepath"

	"github.com/netdata/go-orchestrator/job/confgroup"
)

type format int

const (
	unknownFormat format = iota
	staticFormat
	sdFormat
)

type (
	staticConfig struct {
		confgroup.Default `yaml:",inline"`
		Jobs              []confgroup.Config `yaml:"jobs"`
	}
	sdConfig []confgroup.Config
)

type Reader struct {
	reg   confgroup.Registry
	paths []string
}

func NewReader(reg confgroup.Registry, paths []string) *Reader {
	return &Reader{
		reg:   reg,
		paths: paths,
	}
}

func (r Reader) String() string {
	return "file reader"
}

func (r Reader) Run(ctx context.Context, in chan<- []*confgroup.Group) {
	select {
	case <-ctx.Done():
	case in <- r.groups():
	}
	close(in)
}

func (r Reader) groups() (groups []*confgroup.Group) {
	for _, pattern := range r.paths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			if fi, err := os.Stat(path); err != nil || !fi.Mode().IsRegular() {
				continue
			}

			group, err := parseFile(r.reg, path)
			if err != nil {
				continue
			}
			groups = append(groups, group)
		}
	}
	return groups
}
