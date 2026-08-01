package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"

	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/demo"
	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/task"
)

type Stats struct {
	Queued    int            `json:"queued"`
	Running   int            `json:"running"`
	Completed int            `json:"completed"`
	Failed    int            `json:"failed"`
	Canceled  int            `json:"canceled"`
	Dead      int            `json:"dead"`
	Total     int            `json:"total"`
	Workers   int            `json:"workers"`
	Kind      string         `json:"kind,omitempty"`
	ByKind    map[string]int `json:"by_kind,omitempty"`
}

type ListOptions struct {
	Status task.Status
	Kind   string
	Page   int
	Limit  int
}

type ListResult struct {
	Tasks []task.Task `json:"tasks"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

type Queue struct {
	mu          sync.RWMutex
	tasks       map[string]*task.Task
	order       []string
	ready       chan string
	workers     int
	handler     func(ctx context.Context, t *task.Task) error
	cancelFuncs map[string]context.CancelFunc
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

func New(workers int) *Queue {
	if workers < 1 {
		workers = 2
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		tasks:       make(map[string]*task.Task),
		order:       make([]string, 0),
		ready:       make(chan string, 256),
		workers:     workers,
		handler:     demo.Handle,
		cancelFuncs: make(map[string]context.CancelFunc),
		ctx:         ctx,
		cancel:      cancel,
	}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.workerLoop(i)
	}
	return q
}

func (q *Queue) Workers() int {
	return q.workers
}

func (q *Queue) Stop() {
	q.cancel()
	q.wg.Wait()
}

func (q *Queue) Enqueue(req task.CreateRequest) (task.Task, error) {
	if err := req.Validate(); err != nil {
		return task.Task{}, err
	}
	kind := demo.NormalizeKind(req.Kind)
	if kind == "" {
		return task.Task{}, fmt.Errorf("kind is required (email, media, reports, webhooks, cleanup)")
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}

	now := time.Now().UTC()
	t := &task.Task{
		ID:          newID(),
		Kind:        kind,
		Name:        req.Name, // trimmed by Validate
		Payload:     req.Payload,
		Status:      task.StatusQueued,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	q.mu.Lock()
	q.tasks[t.ID] = t
	q.order = append(q.order, t.ID)
	q.mu.Unlock()

	select {
	case q.ready <- t.ID:
	default:
		go func(id string) { q.ready <- id }(t.ID)
	}

	return *t, nil
}

func (q *Queue) Get(id string) (task.Task, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	t, ok := q.tasks[id]
	if !ok {
		return task.Task{}, false
	}
	return *t, true
}

func (q *Queue) List(opts ListOptions) ListResult {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit < 1 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	filtered := make([]task.Task, 0)
	for i := len(q.order) - 1; i >= 0; i-- {
		t := q.tasks[q.order[i]]
		if opts.Kind != "" && t.Kind != opts.Kind {
			continue
		}
		if opts.Status != "" && t.Status != opts.Status {
			continue
		}
		filtered = append(filtered, *t)
	}

	total := len(filtered)
	start := (opts.Page - 1) * opts.Limit
	if start > total {
		start = total
	}
	end := start + opts.Limit
	if end > total {
		end = total
	}

	return ListResult{
		Tasks: filtered[start:end],
		Total: total,
		Page:  opts.Page,
		Limit: opts.Limit,
	}
}

func (q *Queue) Stats(kind string) Stats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var s Stats
	s.Workers = q.workers
	s.Kind = kind
	s.ByKind = map[string]int{}

	for _, t := range q.tasks {
		s.ByKind[t.Kind]++
		if kind != "" && t.Kind != kind {
			continue
		}
		s.Total++
		switch t.Status {
		case task.StatusQueued:
			s.Queued++
		case task.StatusRunning:
			s.Running++
		case task.StatusCompleted:
			s.Completed++
		case task.StatusFailed:
			s.Failed++
		case task.StatusCanceled:
			s.Canceled++
		case task.StatusDead:
			s.Dead++
		}
	}
	return s
}

func (q *Queue) Cancel(id string) (task.Task, error) {
	q.mu.Lock()
	t, ok := q.tasks[id]
	if !ok {
		q.mu.Unlock()
		return task.Task{}, fmt.Errorf("task not found")
	}
	switch t.Status {
	case task.StatusCompleted, task.StatusCanceled, task.StatusDead:
		q.mu.Unlock()
		return task.Task{}, fmt.Errorf("task cannot be canceled in status %s", t.Status)
	case task.StatusRunning:
		cancelFn, has := q.cancelFuncs[id]
		now := time.Now().UTC()
		t.Status = task.StatusCanceled
		t.UpdatedAt = now
		t.FinishedAt = &now
		q.mu.Unlock()
		if has {
			cancelFn()
		}
		return *t, nil
	default:
		now := time.Now().UTC()
		t.Status = task.StatusCanceled
		t.UpdatedAt = now
		t.FinishedAt = &now
		cp := *t
		q.mu.Unlock()
		return cp, nil
	}
}

func (q *Queue) SeedDemo() {
	for _, d := range demo.SeedRequests() {
		_, _ = q.Enqueue(d)
	}
}

func (q *Queue) workerLoop(id int) {
	defer q.wg.Done()
	for {
		select {
		case <-q.ctx.Done():
			return
		case taskID := <-q.ready:
			q.process(taskID)
		}
	}
}

func (q *Queue) process(id string) {
	q.mu.Lock()
	t, ok := q.tasks[id]
	if !ok || t.Status != task.StatusQueued {
		q.mu.Unlock()
		return
	}
	now := time.Now().UTC()
	t.Status = task.StatusRunning
	t.Attempts++
	t.UpdatedAt = now
	t.StartedAt = &now
	t.LastError = ""
	t.Result = ""
	runCtx, cancel := context.WithCancel(q.ctx)
	q.cancelFuncs[id] = cancel
	snapshot := *t
	q.mu.Unlock()

	err := q.handler(runCtx, &snapshot)

	q.mu.Lock()
	delete(q.cancelFuncs, id)
	cancel()
	current, still := q.tasks[id]
	if !still {
		q.mu.Unlock()
		return
	}
	if current.Status == task.StatusCanceled {
		q.mu.Unlock()
		return
	}

	finished := time.Now().UTC()
	current.UpdatedAt = finished

	if err != nil {
		current.LastError = err.Error()
		if runCtx.Err() != nil {
			current.Status = task.StatusCanceled
			current.FinishedAt = &finished
			q.mu.Unlock()
			return
		}
		if current.Attempts >= current.MaxAttempts {
			current.Status = task.StatusDead
			current.FinishedAt = &finished
			q.mu.Unlock()
			return
		}
		current.Status = task.StatusFailed
		attempts := current.Attempts
		q.mu.Unlock()

		backoff := time.Duration(math.Pow(2, float64(attempts-1))) * 400 * time.Millisecond
		timer := time.NewTimer(backoff)
		defer timer.Stop()
		select {
		case <-q.ctx.Done():
			return
		case <-timer.C:
			q.mu.Lock()
			latest, ok := q.tasks[id]
			if ok && latest.Status == task.StatusFailed {
				latest.Status = task.StatusQueued
				latest.UpdatedAt = time.Now().UTC()
				q.mu.Unlock()
				select {
				case q.ready <- id:
				default:
					go func() { q.ready <- id }()
				}
				return
			}
			q.mu.Unlock()
		}
		return
	}

	current.Status = task.StatusCompleted
	current.FinishedAt = &finished
	current.LastError = ""
	current.Result = snapshot.Result
	q.mu.Unlock()
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "tsk_" + hex.EncodeToString(b[:])
}
