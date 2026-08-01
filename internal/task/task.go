package task

import (
	"fmt"
	"strings"
	"time"
)

// Status is the lifecycle state of a queued job.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
	StatusDead      Status = "dead"
)

// ValidStatuses lists every allowed Status value.
var ValidStatuses = []Status{
	StatusQueued,
	StatusRunning,
	StatusCompleted,
	StatusFailed,
	StatusCanceled,
	StatusDead,
}

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	for _, v := range ValidStatuses {
		if s == v {
			return true
		}
	}
	return false
}

func (s Status) String() string {
	return string(s)
}

// Task is one unit of work in the distributed queue.
type Task struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	Name        string     `json:"name"`
	Payload     string     `json:"payload"`
	Status      Status     `json:"status"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	LastError   string     `json:"last_error,omitempty"`
	Result      string     `json:"result,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

// CanCancel reports whether a task in this status may still be canceled.
func (t Task) CanCancel() bool {
	switch t.Status {
	case StatusQueued, StatusRunning, StatusFailed:
		return true
	default:
		return false
	}
}

// CreateRequest is the input for enqueueing a new task.
type CreateRequest struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Payload     string `json:"payload"`
	MaxAttempts int    `json:"max_attempts"`
}

// Validate trims Name and checks required fields and limits.
// MaxAttempts of 0 is allowed (callers typically default it to 3).
func (r *CreateRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("request is required")
	}
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) > 120 {
		return fmt.Errorf("name must be 120 characters or fewer")
	}
	if len(r.Payload) > 8<<10 {
		return fmt.Errorf("payload must be 8KB or fewer")
	}
	if r.MaxAttempts < 0 {
		return fmt.Errorf("max_attempts cannot be negative")
	}
	if r.MaxAttempts > 20 {
		return fmt.Errorf("max_attempts cannot exceed 20")
	}
	return nil
}

// ParseStatus maps a raw string to a Status (case-insensitive).
func ParseStatus(raw string) (Status, bool) {
	s := Status(strings.ToLower(strings.TrimSpace(raw)))
	if !s.Valid() {
		return "", false
	}
	return s, true
}
