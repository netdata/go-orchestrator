package orchestrator

import (
	"time"
)

func (o *Orchestrator) jobStartLoop() {
	started := make(map[string]bool)
LOOP:
	for {
		select {
		case <-o.jobStartLoopStop:
			break LOOP
		case job := <-o.jobStartCh:
			o.startOnce(job, started)
		}
	}
}

func (o *Orchestrator) startOnce(job Job, started map[string]bool) {
	if started[job.FullName()] {
		log.Infof("skipping %s[%s]: already served by another job", job.ModuleName(), job.Name())
		return
	}

	if o.initJob(job) && o.checkJob(job) && o.postCheckJob(job) {
		started[job.FullName()] = true
		go job.Start()
		o.loopQueue.add(job)
	}

}

func (o *Orchestrator) initJob(job Job) bool {
	if !job.Init() {
		log.Errorf("%s[%s] Init failed", job.ModuleName(), job.Name())
		return false
	}
	return true
}

func (o *Orchestrator) checkJob(job Job) bool {
	ok := job.Check()

	if job.Panicked() {
		return false
	}

	if !ok {
		log.Errorf("%s[%s] Check failed", job.ModuleName(), job.Name())
		if job.AutoDetectionRetry() > 0 {
			go recheckTask(o.jobStartCh, job)
		}
		return false
	}
	return true
}

func (o *Orchestrator) postCheckJob(job Job) bool {
	if !job.PostCheck() {
		log.Errorf("%s[%s] PostCheck failed", job.ModuleName(), job.Name())
		return false
	}
	return true
}

func recheckTask(ch chan Job, job Job) {
	log.Infof("%s[%s] scheduling next check in %d seconds",
		job.ModuleName(),
		job.Name(),
		job.AutoDetectionRetry(),
	)
	time.Sleep(time.Second * time.Duration(job.AutoDetectionRetry()))

	t := time.NewTimer(time.Second * 30)
	defer t.Stop()

	select {
	case <-t.C:
	case ch <- job:
	}
}
