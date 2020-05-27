package module

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testPluginName = "pluginTest"
	testModName    = "modName"
	testJobName    = "jobName"
)

func newTestJob() *Job {
	return NewJob(
		testPluginName,
		testModName,
		nil,
		ioutil.Discard,
	)
}

func TestNewJob(t *testing.T) {
	assert.IsType(t, (*Job)(nil), newTestJob())
}

func TestJob_FullName(t *testing.T) {
	job := newTestJob()

	assert.Equal(t, job.FullName(), testModName)
	job.name = testModName
	assert.Equal(t, job.FullName(), testModName)
	job.name = testJobName
	assert.Equal(t, job.FullName(), fmt.Sprintf("%s_%s", testModName, testJobName))

}

func TestJob_ModuleName(t *testing.T) {
	job := newTestJob()

	assert.Equal(t, job.ModuleName(), testModName)
}

func TestJob_Name(t *testing.T) {
	job := newTestJob()

	assert.Equal(t, job.Name(), testModName)
	job.name = testJobName
	assert.Equal(t, job.Name(), testJobName)
}

func TestJob_Panicked(t *testing.T) {
	job := newTestJob()

	assert.Equal(t, job.Panicked(), job.panicked)
	job.panicked = true
	assert.Equal(t, job.Panicked(), job.panicked)
}

func TestJob_AutoDetectionEvery(t *testing.T) {
	job := newTestJob()

	assert.Equal(t, job.AutoDetectionEvery(), job.AutoDetectEvery)
}

func TestJob_RetryAutoDetection(t *testing.T) {
	job := newTestJob()
	m := &MockModule{
		InitFunc: func() bool {
			return true
		},
		CheckFunc: func() bool { return false },
		ChartsFunc: func() *Charts {
			return &Charts{}
		},
	}
	job.module = m
	job.AutoDetectEvery = 1

	assert.True(t, job.RetryAutoDetection())
	assert.Equal(t, infTries, job.AutoDetectTries)
	for i := 0; i < 1000; i++ {
		job.check()
	}
	assert.True(t, job.RetryAutoDetection())
	assert.Equal(t, infTries, job.AutoDetectTries)

	job.AutoDetectTries = 10
	for i := 0; i < 10; i++ {
		job.check()
	}
	assert.False(t, job.RetryAutoDetection())
	assert.Equal(t, 0, job.AutoDetectTries)
}

func TestJob_AutoDetection(t *testing.T) {
	job := newTestJob()
	var v int
	m := &MockModule{
		InitFunc: func() bool {
			v++
			return true
		},
		CheckFunc: func() bool {
			v++
			return true
		},
		ChartsFunc: func() *Charts {
			v++
			return &Charts{}
		},
	}
	job.module = m

	assert.True(t, job.AutoDetection())
	assert.Equal(t, 3, v)
}

func TestJob_AutoDetection_FailInit(t *testing.T) {
	job := newTestJob()
	m := &MockModule{
		InitFunc: func() bool {
			return false
		},
	}
	job.module = m

	assert.False(t, job.AutoDetection())
	assert.True(t, m.CleanupDone)
}

func TestJob_AutoDetection_FailCheck(t *testing.T) {
	job := newTestJob()
	m := &MockModule{
		InitFunc: func() bool {
			return true
		},
		CheckFunc: func() bool {
			return false
		},
	}
	job.module = m

	assert.False(t, job.AutoDetection())
	assert.True(t, m.CleanupDone)
}

func TestJob_AutoDetection_FailPostCheck(t *testing.T) {
	job := newTestJob()
	m := &MockModule{
		InitFunc: func() bool {
			return true
		},
		CheckFunc: func() bool {
			return true
		},
		ChartsFunc: func() *Charts {
			return nil
		},
	}
	job.module = m

	assert.False(t, job.AutoDetection())
	assert.True(t, m.CleanupDone)
}

func TestJob_AutoDetection_PanicInit(t *testing.T) {
	job := newTestJob()
	m := &MockModule{
		InitFunc: func() bool {
			panic("panic in Init")
		},
	}
	job.module = m

	assert.False(t, job.AutoDetection())
	assert.True(t, m.CleanupDone)
}

func TestJob_AutoDetection_PanicCheck(t *testing.T) {
	job := newTestJob()
	m := &MockModule{
		InitFunc: func() bool {
			return true
		},
		CheckFunc: func() bool {
			panic("panic in Check")
		},
	}
	job.module = m

	assert.False(t, job.AutoDetection())
	assert.True(t, m.CleanupDone)
}

func TestJob_AutoDetection_PanicPostCheck(t *testing.T) {
	job := newTestJob()
	m := &MockModule{
		InitFunc: func() bool {
			return true
		},
		CheckFunc: func() bool {
			return true
		},
		ChartsFunc: func() *Charts {
			panic("panic in PostCheck")
		},
	}
	job.module = m

	assert.False(t, job.AutoDetection())
	assert.True(t, m.CleanupDone)
}

func TestJob_MainLoop(t *testing.T) {
	m := &MockModule{
		ChartsFunc: func() *Charts {
			return &Charts{
				&Chart{
					ID:    "id",
					Title: "title",
					Units: "units",
					Dims: Dims{
						{ID: "id1"},
						{ID: "id2"},
					},
				},
			}
		},
		CollectFunc: func() map[string]int64 {
			return map[string]int64{
				"id1": 1,
				"id2": 2,
			}
		},
	}
	job := newTestJob()
	job.module = m
	job.charts = job.module.Charts()
	job.updateEvery = 1

	go func() {
		for i := 1; i < 3; i++ {
			job.Tick(i)
			time.Sleep(time.Second)
		}
		job.Stop()
	}()

	job.Start()

	assert.True(t, m.CleanupDone)
}

func TestJob_MainLoop_Panic(t *testing.T) {
	m := &MockModule{
		CollectFunc: func() map[string]int64 {
			panic("panic in Collect")
		},
	}
	job := newTestJob()
	job.module = m
	job.updateEvery = 1

	go func() {
		for i := 1; i < 3; i++ {
			time.Sleep(time.Second)
			job.Tick(i)
		}
		job.Stop()
	}()

	job.Start()

	assert.True(t, job.Panicked())
	assert.True(t, m.CleanupDone)
}

func TestJob_Tick(t *testing.T) {
	job := newTestJob()
	for i := 0; i < 3; i++ {
		job.Tick(i)
	}
}

func TestJob_Start(t *testing.T) {
	job := newTestJob()
	job.module = &MockModule{}

	go func() {
		time.Sleep(time.Second)
		job.Stop()
	}()

	job.Start()
}
