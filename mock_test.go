package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestMockJob_Panicked(t *testing.T) {
	m := &mockJob{}

	assert.False(t, m.Panicked())
	m.panicked = func() bool { return true }
	assert.True(t, m.Panicked())
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
