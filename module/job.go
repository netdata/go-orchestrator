package module

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/netdata/go-orchestrator/pkg/logger"
)

func newRuntimeChart(pluginName string) *Chart {
	return &Chart{
		typeID: "netdata",
		Units:  "ms",
		Fam:    pluginName,
		Ctx:    "netdata.go_plugin_execution_time", Priority: 145000,
		Dims: Dims{
			{ID: "time"},
		},
	}
}

type JobConfig struct {
	PluginName      string
	Name            string
	ModuleName      string
	FullName        string
	Module          Module
	Out             io.Writer
	UpdateEvery     int
	AutoDetectEvery int
	Priority        int
}

const (
	penaltyStep = 5
	maxPenalty  = 600
	infTries    = -1
)

func NewJob(cfg JobConfig) *Job {
	var buf bytes.Buffer
	return &Job{
		pluginName:      cfg.PluginName,
		name:            cfg.Name,
		moduleName:      cfg.ModuleName,
		fullName:        cfg.FullName,
		updateEvery:     cfg.UpdateEvery,
		AutoDetectEvery: cfg.AutoDetectEvery,
		priority:        cfg.Priority,
		module:          cfg.Module,
		out:             cfg.Out,
		AutoDetectTries: infTries,
		runtimeChart:    newRuntimeChart(cfg.PluginName),
		stop:            make(chan struct{}, 1),
		tick:            make(chan int),
		buf:             &buf,
		apiWriter:       apiWriter{Writer: &buf},
	}
}

// Job represents a job. It's a module wrapper.
type Job struct {
	pluginName string
	name       string
	moduleName string
	fullName   string

	updateEvery     int
	AutoDetectEvery int
	AutoDetectTries int
	priority        int

	*logger.Logger

	module Module

	initialized bool
	panicked    bool

	runtimeChart *Chart
	charts       *Charts
	tick         chan int
	out          io.Writer
	buf          *bytes.Buffer
	apiWriter    apiWriter

	retries int
	prevRun time.Time

	stop chan struct{}
}

// FullName returns job full name.
func (j Job) FullName() string {
	return j.fullName
}

// ModuleName returns job module name.
func (j Job) ModuleName() string {
	return j.moduleName
}

// Name returns job name.
func (j Job) Name() string {
	return j.name
}

// Panicked returns 'panicked' flag value.
func (j Job) Panicked() bool {
	return j.panicked
}

// AutoDetectionEvery returns value of AutoDetectEvery.
func (j Job) AutoDetectionEvery() int {
	return j.AutoDetectEvery
}

// RetryAutoDetection returns whether it is needed to retry autodetection.
func (j Job) RetryAutoDetection() bool {
	return j.AutoDetectEvery > 0 && (j.AutoDetectTries == infTries || j.AutoDetectTries > 0)
}

// AutoDetection invokes init, check and postCheck. It handles panic.
func (j *Job) AutoDetection() (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
			j.Errorf("PANIC %v", r)
			j.panicked = true
			j.disableAutoDetection()
		}
		if !ok {
			j.module.Cleanup()
		}
	}()

	if ok = j.init(); !ok {
		j.Error("init failed")
		j.disableAutoDetection()
		return
	}
	if ok = j.check(); !ok {
		j.Error("check failed")
		return
	}
	if ok = j.postCheck(); !ok {
		j.Error("postCheck failed")
		j.disableAutoDetection()
		return
	}
	return true
}

// Tick Tick.
func (j *Job) Tick(clock int) {
	select {
	case j.tick <- clock:
	default:
		j.Debug("skip the tick due to previous run hasn't been finished")
	}
}

// Start starts job main loop.
func (j *Job) Start() {
LOOP:
	for {
		select {
		case <-j.stop:
			break LOOP
		case t := <-j.tick:
			if t%(j.updateEvery+j.penalty()) == 0 {
				j.runOnce()
			}
		}
	}
	j.module.Cleanup()
	j.cleanup()
	j.stop <- struct{}{}
}

// Stop stops job main loop.
func (j *Job) Stop() {
	j.stop <- struct{}{}
	<-j.stop
}

func (j *Job) disableAutoDetection() {
	j.AutoDetectEvery = 0
}

func (j *Job) cleanup() {
	logger.GlobalMsgCountWatcher.Unregister(j.Logger)
	j.buf.Reset()

	if j.runtimeChart.created {
		j.runtimeChart.MarkRemove()
		j.createChart(j.runtimeChart)
	}
	for _, chart := range *j.charts {
		if chart.created {
			chart.MarkRemove()
			j.createChart(chart)
		}
	}
	_, _ = io.Copy(j.out, j.buf)
}

func (j *Job) init() bool {
	if j.initialized {
		return true
	}

	limitedLogger := logger.NewLimited(j.pluginName, j.ModuleName(), j.Name())
	j.Logger = limitedLogger
	j.module.GetBase().Logger = limitedLogger

	j.initialized = j.module.Init()
	return j.initialized
}

func (j *Job) check() bool {
	ok := j.module.Check()
	if !ok && j.AutoDetectTries != infTries {
		j.AutoDetectTries--
	}
	return ok
}

func (j *Job) postCheck() bool {
	if j.charts = j.module.Charts(); j.charts == nil {
		j.Error("nil charts")
		return false
	}
	if err := checkCharts(*j.charts...); err != nil {
		j.Errorf("charts check: %v", err)
		return false
	}
	return true
}

func (j *Job) runOnce() {
	curTime := time.Now()
	sinceLastRun := calcSinceLastRun(curTime, j.prevRun)
	j.prevRun = curTime

	metrics := j.collect()

	if j.panicked {
		return
	}

	if j.processMetrics(metrics, curTime, sinceLastRun) {
		j.retries = 0
	} else {
		j.retries++
	}

	_, _ = io.Copy(j.out, j.buf)
	j.buf.Reset()
}

func (j *Job) collect() (result map[string]int64) {
	j.panicked = false
	defer func() {
		if r := recover(); r != nil {
			j.Errorf("PANIC: %v", r)
			j.panicked = true
		}
	}()
	return j.module.Collect()
}

func (j *Job) processMetrics(metrics map[string]int64, startTime time.Time, sinceLastRun int) bool {
	if !j.runtimeChart.created {
		j.runtimeChart.ID = fmt.Sprintf("execution_time_of_%s", j.FullName())
		j.runtimeChart.Title = fmt.Sprintf("Execution Time for %s", j.FullName())
		j.createChart(j.runtimeChart)
	}

	var (
		remove  []string
		updated int
		elapsed = int64(durationTo(time.Now().Sub(startTime), time.Millisecond))
	)

	for _, chart := range *j.charts {
		if !chart.created {
			j.createChart(chart)
		}
		if chart.remove {
			remove = append(remove, chart.ID)
			continue
		}
		if len(metrics) == 0 || chart.Obsolete {
			continue
		}
		if j.updateChart(chart, metrics, sinceLastRun) {
			updated++
		}
	}

	for _, id := range remove {
		_ = j.charts.Remove(id)
	}

	if updated == 0 {
		return false
	}

	j.updateChart(j.runtimeChart, map[string]int64{"time": elapsed}, sinceLastRun)

	return true
}

func (j *Job) createChart(chart *Chart) {
	if chart.Priority == 0 {
		chart.Priority = j.priority
		j.priority++
	}
	_ = j.apiWriter.chart(
		firstNotEmpty(chart.typeID, j.FullName()),
		chart.ID,
		chart.OverID,
		chart.Title,
		chart.Units,
		chart.Fam,
		chart.Ctx,
		chart.Type,
		chart.Priority,
		j.updateEvery,
		chart.Opts,
		j.pluginName,
		j.moduleName,
	)
	for _, dim := range chart.Dims {
		_ = j.apiWriter.dimension(
			dim.ID,
			dim.Name,
			dim.Algo,
			dim.Mul,
			dim.Div,
			dim.DimOpts,
		)
	}
	for _, v := range chart.Vars {
		_ = j.apiWriter.varSet(
			v.ID,
			v.Value,
		)
	}
	_, _ = j.apiWriter.Write([]byte("\n"))

	chart.created = true
}

func (j *Job) updateChart(chart *Chart, collected map[string]int64, sinceLastRun int) bool {
	if !chart.updated {
		sinceLastRun = 0
	}

	_ = j.apiWriter.begin(
		firstNotEmpty(chart.typeID, j.FullName()),
		chart.ID,
		sinceLastRun,
	)

	var (
		remove  []string
		updated int
	)

	for _, dim := range chart.Dims {
		if dim.remove {
			remove = append(remove, dim.ID)
			continue
		}
		v, ok := collected[dim.ID]

		if !ok {
			_ = j.apiWriter.dimSetEmpty(dim.ID)
		} else {
			_ = j.apiWriter.dimSet(dim.ID, v)
			updated++
		}
	}

	for _, id := range remove {
		_ = chart.RemoveDim(id)
	}

	for _, variable := range chart.Vars {
		v, ok := collected[variable.ID]
		if ok {
			_ = j.apiWriter.varSet(variable.ID, v)
		}
	}

	_ = j.apiWriter.end()

	if chart.updated = updated > 0; chart.updated {
		chart.Retries = 0
	} else {
		chart.Retries++
	}
	return chart.updated
}

func (j Job) penalty() int {
	v := j.retries / penaltyStep * penaltyStep * j.updateEvery / 2
	if v > maxPenalty {
		return maxPenalty
	}
	return v
}

func calcSinceLastRun(curTime, prevRun time.Time) int {
	if prevRun.IsZero() {
		return 0
	}
	// monotonic
	// durationTo(curTime.Sub(prevRun), time.Microsecond)
	return int((curTime.UnixNano() - prevRun.UnixNano()) / 1000)
}

func durationTo(duration time.Duration, to time.Duration) int {
	return int(int64(duration) / (int64(to) / int64(time.Nanosecond)))
}

func firstNotEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
