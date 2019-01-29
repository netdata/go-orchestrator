package orchestrator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_initJob(t *testing.T) {
	o := New()
	job := &mockJob{}

	// OK case
	job.init = func() bool { return true }
	assert.True(t, o.initJob(job))

	// NG case
	job.init = func() bool { return false }
	assert.False(t, o.initJob(job))
}

func Test_jobCheck(t *testing.T) {
	o := New()

	job := &mockJob{}

	// OK case
	job.check = func() bool { return true }
	assert.True(t, o.checkJob(job))

	// NG case
	job.check = func() bool { return false }
	assert.False(t, o.checkJob(job))

	// Panic case
	job.check = func() bool { return true }
	job.panicked = func() bool { return true }
	assert.False(t, o.checkJob(job))

	// AutoDetectionRetry case
	job.check = func() bool { return false }
	job.panicked = func() bool { return false }
	job.autoDetectionRetry = func() int { return 1 }
	assert.False(t, o.checkJob(job))

	wait := time.NewTimer(time.Second * 2)
	defer wait.Stop()

	select {
	case <-wait.C:
		t.Error("auto detection retry test failed")
	case <-o.jobStartCh:
	}
}

func Test_jobPostCheck(t *testing.T) {
	o := New()

	job := &mockJob{}

	// OK case
	job.postCheck = func() bool { return true }
	assert.True(t, o.postCheckJob(job))

	// NG case
	job.postCheck = func() bool { return false }
	assert.False(t, o.postCheckJob(job))
}

func Test_jobStartLoop(t *testing.T) {
	o := New()

	go o.jobStartLoop()

	job := &mockJob{}

	o.jobStartCh <- job
	o.jobStartCh <- job
	o.jobStartCh <- job
	o.jobStartLoopStop <- struct{}{}

	assert.Equal(t, 1, len(o.loopQueue.queue))

	for _, j := range o.loopQueue.queue {
		j.Stop()
	}
}
