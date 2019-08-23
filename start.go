package orchestrator

import (
	"time"
)

func (o *Orchestrator) jobStartLoop() {
	started := make(map[string]bool)
LOOP:
	for {
		select {
		case <-o.jobStartStop:
			break LOOP
		case job := <-o.jobStartCh:
			o.startOnce(job, started)
		}
	}
}

func (o *Orchestrator) startOnce(job Job, started map[string]bool) {
	if started[job.FullName()] {
		log.Warningf("%s[%s]: already served by another job, skipping ", job.ModuleName(), job.Name())
		o.jobsStatuses.remove(job)
		return
	}

	ok := job.AutoDetection()
	if job.Panicked() {
		log.Errorf("%s[%s]: panic during autodetection, skipping", job.ModuleName(), job.Name())
		o.jobsStatuses.remove(job)
		return
	}

	if !ok && job.RetryAutoDetection() {
		go recheckTask(o.jobStartCh, job)
		o.jobsStatuses.put(job, "recovering")
		return
	}

	if !ok {
		log.Warningf("%s[%s]: autodetection failed", job.ModuleName(), job.Name())
		o.jobsStatuses.remove(job)
		return
	}

	log.Infof("%s[%s]: autodetection success", job.ModuleName(), job.Name())

	started[job.FullName()] = true
	o.jobsStatuses.put(job, "active")
	go job.Start()
	o.loopQueue.add(job)
}

func recheckTask(ch chan Job, job Job) {
	log.Infof("%s[%s]: scheduling next check in %d seconds",
		job.ModuleName(),
		job.Name(),
		job.AutoDetectionEvery(),
	)
	time.Sleep(time.Second * time.Duration(job.AutoDetectionEvery()))

	t := time.NewTimer(time.Second * 30)
	defer t.Stop()

	select {
	case <-t.C:
	case ch <- job:
	}
}
