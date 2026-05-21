package types

import "time"

type Source struct {
	Name string
	Path string
}

type Dimension struct {
	Number  string
	Name    string
	Title   string
	File    string
}

type TaskState struct {
	DimensionNumber  string     `json:"dimensionNumber"`
	DimensionName    string     `json:"dimensionName"`
	DimensionTitle   string     `json:"dimensionTitle"`
	SourceName       string     `json:"sourceName"`
	Status           string     `json:"status"`
	Attempts         int        `json:"attempts"`
	LastError        string     `json:"lastError,omitempty"`
	LastAttemptAt    *time.Time `json:"lastAttemptAt,omitempty"`
	NextRetryAt      *time.Time `json:"nextRetryAt,omitempty"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

type SynthesisState struct {
	DimensionNumber string     `json:"dimensionNumber"`
	DimensionName    string     `json:"dimensionName"`
	DimensionTitle   string     `json:"dimensionTitle"`
	Status           string     `json:"status"`
	Attempts         int        `json:"attempts"`
	LastError        string     `json:"lastError,omitempty"`
	LastAttemptAt    *time.Time `json:"lastAttemptAt,omitempty"`
	NextRetryAt      *time.Time `json:"nextRetryAt,omitempty"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

type RunState struct {
	Version     int               `json:"version"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	BatchSize   int                `json:"batchSize"`
	Tasks       []TaskState        `json:"tasks"`
	SynthesisTasks []SynthesisState `json:"synthesisTasks"`
	IsComplete  bool              `json:"isComplete"`
}

const (
	StatusPending    = "pending"
	StatusRunning    = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)