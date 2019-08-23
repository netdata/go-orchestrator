package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_jobStartLoop(t *testing.T) {
	o := New()

	go o.jobStartLoop()

	job := &mockJob{}

	o.jobStartCh <- job
	o.jobStartCh <- job
	o.jobStartCh <- job
	o.jobStartStop <- struct{}{}

	assert.Equal(t, 1, len(o.loopQueue.queue))
	assert.Equal(t, 1, len(o.jobsStatuses.items))

	for _, j := range o.loopQueue.queue {
		j.Stop()
	}
}
