package file

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"gopkg.in/yaml.v2"
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

			group, err := readFile(r.reg, path)
			if err != nil {
				continue
			}
			groups = append(groups, group)
		}
	}
	return groups
}

func readFile(req confgroup.Registry, path string) (*confgroup.Group, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, err
	}

	switch cfgFormat(bs) {
	case staticFormat:
		return readStaticFormat(req, path, bs)
	case sdFormat:
		return readSDFormat(req, path, bs)
	default:
		return nil, fmt.Errorf("unknown file format ('%s')", path)
	}
}

func readStaticFormat(reg confgroup.Registry, path string, bs []byte) (*confgroup.Group, error) {
	name := fileName(path)
	modDef, ok := reg.Lookup(name)
	if !ok {
		return nil, nil
	}

	var modCfg staticConfig
	if err := yaml.Unmarshal(bs, &modCfg); err != nil {
		return nil, err
	}
	for _, cfg := range modCfg.Jobs {
		cfg.Set("module", name)
		def := mergeDef(modCfg.Default, modDef)
		cfg.Apply(def)
	}
	group := &confgroup.Group{
		Configs: modCfg.Jobs,
		Source:  path,
	}
	return group, nil
}

func readSDFormat(reg confgroup.Registry, path string, bs []byte) (*confgroup.Group, error) {
	var cfgs sdConfig
	if err := yaml.Unmarshal(bs, &cfgs); err != nil {
		return nil, err
	}

	var i int
	for _, cfg := range cfgs {
		if def, ok := reg.Lookup(cfg.Module()); ok {
			cfg.Apply(def)
			cfgs[i] = cfg
			i++
		}
	}

	group := &confgroup.Group{
		Configs: cfgs[:i],
		Source:  path,
	}
	return group, nil
}

func cfgFormat(bs []byte) format {
	var data interface{}
	if err := yaml.Unmarshal(bs, &data); err != nil {
		return unknownFormat
	}
	type (
		static = map[interface{}]interface{}
		sd     = []interface{}
	)
	switch data.(type) {
	case static:
		return staticFormat
	case sd:
		return sdFormat
	default:
		return unknownFormat
	}
}

func mergeDef(a, b confgroup.Default) confgroup.Default {
	return confgroup.Default{
		MinUpdateEvery:     firstPositive(a.MinUpdateEvery, b.MinUpdateEvery),
		UpdateEvery:        firstPositive(a.UpdateEvery, b.UpdateEvery),
		AutoDetectionRetry: firstPositive(a.AutoDetectionRetry, b.AutoDetectionRetry),
		Priority:           firstPositive(a.Priority, b.Priority),
	}
}

func firstPositive(value int, others ...int) int {
	if value > 0 || len(others) == 0 {
		return value
	}
	return firstPositive(others[0], others[1:]...)
}

func fileName(path string) string {
	_, file := filepath.Split(path)
	ext := filepath.Ext(path)
	return file[:len(file)-len(ext)]
}
