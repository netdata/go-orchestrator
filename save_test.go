package orchestrator

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_jobStatusesSaveLoop(t *testing.T) {
	b := &bytes.Buffer{}
	js := newJobsStatuses()
	s := newJobsStatusesSaver(b, js, 1)

	js.put(&mockJob{}, "active")
	s.runOnce()
	go func() {
		time.Sleep(time.Second * 3)
		s.stop()
	}()
	s.mainLoop()

	assert.NotZero(t, b.Bytes())
}
