package run

import (
	"context"
	"sync"
	"time"

	"github.com/netdata/go-orchestrator/job"
	"github.com/netdata/go-orchestrator/pkg/logger"
	"github.com/netdata/go-orchestrator/pkg/ticker"
)

type (
	Manager struct {
		*logger.Logger
		mux   sync.Mutex
		queue queue
	}
	queue []job.Job
)

func NewManager() *Manager {
	return &Manager{
		mux: sync.Mutex{},
	}
}

func (m *Manager) Run(ctx context.Context) {
	tk := ticker.New(time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case clock := <-tk.C:
			m.notify(clock)
		}
	}
}

// Starts starts a job and adds it to the job queue.
func (m *Manager) Start(job job.Job) {
	m.mux.Lock()
	defer m.mux.Unlock()

	go job.Start()
	m.queue.add(job)
}

// Stop stops a job and removes it from the job queue.
func (m *Manager) Stop(fullName string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if j := m.queue.remove(fullName); j != nil {
		j.Stop()
	}
}

// Cleanup stops all jobs in the queue.
func (m *Manager) Cleanup() {
	for _, v := range m.queue {
		v.Stop()
	}
	m.queue = m.queue[:0]
}

func (m *Manager) notify(clock int) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, v := range m.queue {
		v.Tick(clock)
	}
}

func (q *queue) add(job job.Job) {
	*q = append(*q, job)
}

func (q *queue) remove(fullName string) job.Job {
	for idx, v := range *q {
		if v.FullName() != fullName {
			continue
		}
		j := (*q)[idx]
		copy((*q)[idx:], (*q)[idx+1:])
		(*q)[len(*q)-1] = nil
		*q = (*q)[:len(*q)-1]
		return j
	}
	return nil
}
