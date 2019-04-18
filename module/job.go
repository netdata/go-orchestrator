package module

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/netdata/go-orchestrator/logger"
)

const (
	penaltyStep = 5
	maxPenalty  = 600
)

// NewJob returns new job.
func NewJob(pluginName string, moduleName string, module Module, out io.Writer) *Job {
	buf := &bytes.Buffer{}
	return &Job{
		pluginName: pluginName,
		moduleName: moduleName,
		module:     module,
		out:        out,
		runtimeChart: &Chart{
			typeID: "netdata",
			Units:  "ms",
			Fam:    pluginName,
			Ctx:    "netdata.go_plugin_execution_time", Priority: 145000,
			Dims: Dims{
				{ID: "time"},
			},
		},
		stopHook:  make(chan struct{}, 1),
		tick:      make(chan int),
		buf:       buf,
		apiWriter: apiWriter{Writer: buf},
	}
}

// Job represents a job. It's a module wrapper.
type Job struct {
	Nam             string `yaml:"name"`
	UpdateEvery     int    `yaml:"update_every"`
	AutoDetectRetry int    `yaml:"autodetection_retry"`
	Priority        int    `yaml:"priority"`

	*logger.Logger

	pluginName string
	moduleName string
	module     Module

	initialized bool
	panicked    bool

	stopHook     chan struct{}
	runtimeChart *Chart
	charts       *Charts
	tick         chan int
	out          io.Writer
	buf          *bytes.Buffer
	apiWriter    apiWriter

	retries int
	prevRun time.Time
}

// FullName returns full name.
// If name isn't specified or equal to module name it returns module name.
func (j Job) FullName() string {
	if j.Nam == "" || j.Nam == j.moduleName {
		return j.ModuleName()
	}
	return fmt.Sprintf("%s_%s", j.ModuleName(), j.Name())
}

// ModuleName returns module name.
func (j Job) ModuleName() string {
	return j.moduleName
}

// Name returns name.
// If name isn't specified it returns module name.
func (j Job) Name() string {
	if j.Nam == "" {
		return j.moduleName
	}
	return j.Nam
}

// Panicked returns 'panicked' flag value.
func (j Job) Panicked() bool {
	return j.panicked
}

// Init calls module Init and returns its value.
func (j *Job) Init() bool {
	if j.initialized {
		return true
	}

	defer func() {
		if r := recover(); r != nil {
			j.Errorf("PANIC %v", r)
			j.panicked = true
			j.module.Cleanup()
		}
	}()

	limitedLogger := logger.NewLimited(j.pluginName, j.ModuleName(), j.Name())
	j.Logger = limitedLogger
	j.module.SetLogger(limitedLogger)

	ok := j.module.Init()
	if ok {
		j.initialized = true
	}

	return ok
}

// Check calls module Check and returns its value.
// It handles panic. In case of panic it calls module Cleanup.
func (j *Job) Check() (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
			j.Errorf("PANIC %v", r)
			j.panicked = true
			j.module.Cleanup()
		}
	}()
	ok = j.module.Check()
	if !ok {
		j.module.Cleanup()
	}
	return
}

// PostCheck calls module Charts.
// If the result is nil it calls module Cleanup.
func (j *Job) PostCheck() bool {
	j.charts = j.module.Charts()

	if j.charts == nil {
		j.Error("Charts can't be nil")
		j.module.Cleanup()
		return false
	}

	return true
}

// Tick Tick
func (j *Job) Tick(clock int) {
	select {
	case j.tick <- clock:
	default:
		j.Errorf("Skip the tick due to previous run hasn't been finished.")
	}
}

// Start simply invokes MainLoop.
func (j *Job) Start() {
	j.MainLoop()
}

// Stop stops MainLoop
func (j *Job) Stop() {
	j.stopHook <- struct{}{}
}

// MainLoop is a job main function.
func (j *Job) MainLoop() {
LOOP:
	for {
		select {
		case <-j.stopHook:
			j.module.Cleanup()
			break LOOP
		case t := <-j.tick:
			doRun := t%(j.UpdateEvery+j.penalty()) == 0
			if doRun {
				j.runOnce()
			}
		}
	}
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

// AutoDetectionRetry returns value of AutoDetectRetry.
func (j Job) AutoDetectionRetry() int {
	return j.AutoDetectRetry
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
		remove       []string
		totalUpdated int
		elapsed      = int64(durationTo(time.Now().Sub(startTime), time.Millisecond))
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
			totalUpdated++
		}

	}

	for _, id := range remove {
		_ = j.charts.Remove(id)
	}

	if totalUpdated == 0 {
		return false
	}

	j.updateChart(j.runtimeChart, map[string]int64{"time": elapsed}, sinceLastRun)

	return true
}

func (j *Job) createChart(chart *Chart) {
	if chart.Priority == 0 {
		chart.Priority = j.Priority
		j.Priority++
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
		j.UpdateEvery,
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

func (j *Job) updateChart(chart *Chart, data map[string]int64, sinceLastRun int) bool {
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
		v, ok := data[dim.ID]

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
		v, ok := data[variable.ID]
		if ok {
			_ = j.apiWriter.varSet(variable.ID, v)
		}
	}

	_ = j.apiWriter.end()

	chart.updated = updated > 0

	if chart.updated {
		chart.Retries = 0
	} else {
		chart.Retries++
	}

	return chart.updated
}

func (j Job) penalty() int {
	v := j.retries / penaltyStep * penaltyStep * j.UpdateEvery / 2
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
