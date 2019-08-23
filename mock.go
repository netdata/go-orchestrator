package orchestrator

type mockJob struct {
	fullName           func() string
	moduleName         func() string
	name               func() string
	autodetection      func() bool
	autodetectionEvery func() int
	retryAutodetection func() bool
	panicked           func() bool
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

// Panicked return whether the mock job is panicked.
func (m mockJob) Panicked() bool {
	if m.panicked == nil {
		return false
	}
	return m.panicked()
}

// Tick invokes mock job Tick.
func (m mockJob) Tick(clock int) {
	if m.tick == nil {
		return
	}
	m.tick(clock)
}

// Start invokes mock job Start.
func (m mockJob) Start() {
	if m.start == nil {
		return
	}
	m.start()
}

// Stop invokes mock job Stop.
func (m mockJob) Stop() {
	if m.stop == nil {
		return
	}
	m.stop()
}
