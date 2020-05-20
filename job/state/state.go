package state

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"
)

const stateFile = "god-jobs-statuses.json"

var varLibDir = os.Getenv("NETDATA_LIB_DIR")

type Manager struct {
	path  string
	state *State
}

func NewManager() *Manager {
	return &Manager{
		state: &State{mux: new(sync.Mutex)},
		path:  filepath.Join(varLibDir, stateFile),
	}
}

func (w *Manager) Run(ctx context.Context) {
	tk := time.NewTicker(time.Second * 10)
	defer tk.Stop()
	defer w.save()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			w.save()
		}
	}
}

func (w *Manager) Save(cfg confgroup.Config, state string) {
	w.state.add(cfg, state)
}

func (w *Manager) Remove(cfg confgroup.Config) {
	w.state.remove(cfg)
}

func (w *Manager) save() {
	bs, err := w.state.bytes()
	if err != nil {
		return
	}
	f, err := os.Create(w.path)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(bs)
}

type State struct {
	mux   *sync.Mutex
	items map[string]map[string]string
}

func (s State) Contains(cfg confgroup.Config, states ...string) bool {
	state, ok := s.lookup(cfg)
	if !ok {
		return false
	}
	for _, v := range states {
		if state == v {
			return true
		}
	}
	return false
}

func (s *State) lookup(cfg confgroup.Config) (string, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	v, ok := s.items[cfg.Module()]
	if !ok {
		return "", false
	}
	state, ok := v[cfg.Name()]
	return state, ok
}

func (s *State) add(cfg confgroup.Config, state string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.items == nil {
		s.items = make(map[string]map[string]string)
	}
	if s.items[cfg.Module()] == nil {
		s.items[cfg.Module()] = make(map[string]string)
	}
	s.items[cfg.Module()][cfg.Name()] = state
}

func (s *State) remove(cfg confgroup.Config) {
	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.items[cfg.Module()], cfg.Name())
	if len(s.items[cfg.Module()]) == 0 {
		delete(s.items, cfg.Module())
	}
}

func (s *State) bytes() ([]byte, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	return json.MarshalIndent(s.items, "", " ")
}

func Load(path string) (*State, error) {
	state := &State{mux: new(sync.Mutex)}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return state, json.NewDecoder(f).Decode(&state.items)
}
