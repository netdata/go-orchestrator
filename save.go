package orchestrator

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

func newJobsStatusesSaver(w io.Writer, js *jobsStatuses, saveEvery time.Duration) *jobsStatusesSaver {
	return &jobsStatusesSaver{
		Writer: w,
		js:     js,
		freq:   saveEvery,
		once:   sync.Once{},
		stopCh: make(chan struct{}),
	}
}

type jobsStatusesSaver struct {
	io.Writer
	js     *jobsStatuses
	freq   time.Duration
	once   sync.Once
	stopCh chan struct{}
}

func (s *jobsStatusesSaver) mainLoop() {
	t := time.NewTicker(s.freq)
	defer t.Stop()
LOOP:
	for {
		select {
		case <-s.stopCh:
			break LOOP
		case <-t.C:
			s.runOnce()
		}
	}
}

func (s *jobsStatusesSaver) stop() {
	s.once.Do(func() { close(s.stopCh) })
}

func (s *jobsStatusesSaver) runOnce() {
	b, err := s.js.asBytes()
	if err != nil {
		log.Errorf("error on converting jobs statuses : %v", err)
		return
	}
	_, err = s.Write(b)
	if err != nil {
		log.Errorf("error on writing jobs statuses : %v", err)
	}
}

type fileWriter struct {
	path string
}

func (s fileWriter) Write(data []byte) (n int, err error) {
	f, err := os.Create(s.path)
	if err != nil {
		return
	}
	defer f.Close()

	n, err = f.Write(data)
	return
}

func newJobsStatuses() *jobsStatuses {
	return &jobsStatuses{mux: new(sync.Mutex), items: make(map[string]map[string]string)}
}

type jobsStatuses struct {
	mux   *sync.Mutex
	items map[string]map[string]string
}

func (js jobsStatuses) contains(job Job) bool {
	js.mux.Lock()
	defer js.mux.Unlock()

	v, ok := js.items[job.ModuleName()]
	if !ok {
		return false
	}
	_, ok = v[job.Name()]
	return ok
}

func (js *jobsStatuses) put(job Job, status string) {
	js.mux.Lock()
	defer js.mux.Unlock()

	_, ok := js.items[job.ModuleName()]
	if !ok {
		js.items[job.ModuleName()] = make(map[string]string)
	}
	js.items[job.ModuleName()][job.Name()] = status
}

func (js *jobsStatuses) remove(job Job) {
	js.mux.Lock()
	defer js.mux.Unlock()

	delete(js.items[job.ModuleName()], job.Name())
}

func (js *jobsStatuses) asBytes() ([]byte, error) {
	js.mux.Lock()
	defer js.mux.Unlock()

	return json.MarshalIndent(js.items, "", " ")
}

func loadJobsStatusesFromFile(absPath string) (*jobsStatuses, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &jobsStatuses{mux: new(sync.Mutex)}
	if err = json.NewDecoder(f).Decode(&s.items); err != nil {
		return nil, err
	}
	return s, nil
}
