package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type actionJob struct {
	ID        string    `json:"job_id"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	Submitted time.Time `json:"submitted_at"`
	Started   time.Time `json:"started_at,omitzero"`
	Finished  time.Time `json:"finished_at,omitzero"`
	StatusURL string    `json:"status_url"`
}

type actionTask struct {
	jobID string
	run   func(context.Context) (string, error)
}

type actionJobs struct {
	mu     sync.RWMutex
	jobs   map[string]actionJob
	queue  chan actionTask
	nextID atomic.Uint64
	cancel context.CancelFunc
}

const maxRetainedActionJobs = 100

func newActionJobs(parent context.Context) *actionJobs {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	jobs := &actionJobs{
		jobs:   make(map[string]actionJob),
		queue:  make(chan actionTask, 32),
		cancel: cancel,
	}
	go jobs.run(ctx)
	return jobs
}

func (j *actionJobs) submit(action string, task func(context.Context) (string, error)) (actionJob, error) {
	id := strconv.FormatUint(j.nextID.Add(1), 10)
	job := actionJob{
		ID:        id,
		Action:    action,
		Status:    "queued",
		Message:   "Action queued.",
		Submitted: time.Now().UTC(),
		StatusURL: "/actions/" + id,
	}
	j.mu.Lock()
	j.pruneCompletedLocked()
	if len(j.jobs) >= maxRetainedActionJobs {
		j.mu.Unlock()
		return actionJob{}, errors.New("too many active action jobs")
	}
	j.jobs[id] = job
	j.mu.Unlock()
	select {
	case j.queue <- actionTask{jobID: id, run: task}:
		return job, nil
	default:
		j.mu.Lock()
		delete(j.jobs, id)
		j.mu.Unlock()
		return actionJob{}, errors.New("action queue is full")
	}
}

func (j *actionJobs) pruneCompletedLocked() {
	for len(j.jobs) >= maxRetainedActionJobs {
		var oldestID string
		var oldest time.Time
		for id, job := range j.jobs {
			if job.Finished.IsZero() || (!oldest.IsZero() && !job.Finished.Before(oldest)) {
				continue
			}
			oldestID = id
			oldest = job.Finished
		}
		if oldestID == "" {
			return
		}
		delete(j.jobs, oldestID)
	}
}

func (j *actionJobs) get(id string) (actionJob, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	job, ok := j.jobs[id]
	return job, ok
}

func (j *actionJobs) close() {
	if j != nil && j.cancel != nil {
		j.cancel()
	}
}

func (j *actionJobs) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-j.queue:
			j.update(task.jobID, func(job *actionJob) {
				job.Status = "running"
				job.Message = "Action running."
				job.Started = time.Now().UTC()
			})
			message, err := runActionTask(ctx, task.run)
			j.update(task.jobID, func(job *actionJob) {
				job.Finished = time.Now().UTC()
				if err != nil {
					job.Status = "failed"
					job.Message = "Action failed."
					job.Error = err.Error()
					return
				}
				job.Status = "succeeded"
				job.Message = message
			})
		}
	}
}

func runActionTask(ctx context.Context, run func(context.Context) (string, error)) (message string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("action panicked: %v", recovered)
		}
	}()
	return run(ctx)
}

func (j *actionJobs) update(id string, update func(*actionJob)) {
	j.mu.Lock()
	defer j.mu.Unlock()
	job := j.jobs[id]
	update(&job)
	j.jobs[id] = job
}
