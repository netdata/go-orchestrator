package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockJob struct {
	fullName           func() string
	moduleName         func() string
	name               func() string
	autodetection      func() bool
	autodetectionEvery func() int
	retryAutodetection func() bool
	tick               func(int)
	start              func()
	stop               func()
}

// FullName returns mock job full name.
func (m mockJob) FullName() string {
	if m.fullName == nil {
		return "mock"
	}
	return m.fullName()
}

// ModuleName returns mock job module name.
func (m mockJob) ModuleName() string {
	if m.moduleName == nil {
		return "mock"
	}
	return m.moduleName()
}

// Name returns mock job name.
func (m mockJob) Name() string {
	if m.name == nil {
		return "mock"
	}
	return m.name()
}

// AutoDetectionEvery returns mock job AutoDetectionEvery.
func (m mockJob) AutoDetectionEvery() int {
	if m.autodetectionEvery == nil {
		return 0
	}
	return m.autodetectionEvery()
}

// AutoDetection returns mock job AutoDetection.
func (m mockJob) AutoDetection() bool {
	if m.autodetection == nil {
		return true
	}
	return m.autodetection()
}

// RetryAutoDetection invokes mock job RetryAutoDetection.
func (m mockJob) RetryAutoDetection() bool {
	if m.retryAutodetection == nil {
		return true
	}
	return m.retryAutodetection()
}

// Tick invokes mock job Tick.
func (m mockJob) Tick(clock int) {
	if m.tick != nil {
		m.tick(clock)
	}
}

// Start invokes mock job Start.
func (m mockJob) Start() {
	if m.start != nil {
		m.start()
	}
}

// Stop invokes mock job Stop.
func (m mockJob) Stop() {
	if m.stop != nil {
		m.stop()
	}
}

func TestMockJob_FullName(t *testing.T) {
	m := &mockJob{}
	expected := "name"

	assert.NotEqual(t, expected, m.FullName())
	m.fullName = func() string { return expected }
	assert.Equal(t, expected, m.FullName())
}

func TestMockJob_ModuleName(t *testing.T) {
	m := &mockJob{}
	expected := "name"

	assert.NotEqual(t, expected, m.ModuleName())
	m.moduleName = func() string { return expected }
	assert.Equal(t, expected, m.ModuleName())
}

func TestMockJob_Name(t *testing.T) {
	m := &mockJob{}
	expected := "name"

	assert.NotEqual(t, expected, m.Name())
	m.name = func() string { return expected }
	assert.Equal(t, expected, m.Name())
}

func TestMockJob_AutoDetectionEvery(t *testing.T) {
	m := &mockJob{}
	expected := -1

	assert.NotEqual(t, expected, m.AutoDetectionEvery())
	m.autodetectionEvery = func() int { return expected }
	assert.Equal(t, expected, m.AutoDetectionEvery())
}

func TestMockJob_RetryAutoDetection(t *testing.T) {
	m := &mockJob{}
	expected := true

	assert.True(t, m.RetryAutoDetection())
	m.retryAutodetection = func() bool { return expected }
	assert.True(t, m.RetryAutoDetection())
}

func TestMockJob_AutoDetection(t *testing.T) {
	m := &mockJob{}
	expected := true

	assert.True(t, m.AutoDetection())
	m.autodetection = func() bool { return expected }
	assert.True(t, m.AutoDetection())
}

func TestMockJob_Tick(t *testing.T) {
	m := &mockJob{}

	assert.NotPanics(t, func() { m.Tick(1) })
}

func TestMockJob_Start(t *testing.T) {
	m := &mockJob{}

	assert.NotPanics(t, func() { m.Start() })
}

func TestMockJob_Stop(t *testing.T) {
	m := &mockJob{}

	assert.NotPanics(t, func() { m.Stop() })
}
