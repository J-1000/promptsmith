package db

import "time"

// Domain model types persisted by the data layer.

type Project struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Prompt struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	FilePath    string
	CreatedAt   time.Time
}

type PromptWithLatestVersion struct {
	Prompt
	LatestVersion string
}

type PromptVersion struct {
	ID              string
	PromptID        string
	Version         string
	Content         string
	Variables       string // JSON
	Metadata        string // JSON
	ParentVersionID *string
	CommitMessage   string
	CreatedAt       time.Time
	CreatedBy       string
}

type Tag struct {
	ID        string
	PromptID  string
	VersionID string
	Name      string
	CreatedAt time.Time
}

type TestRun struct {
	ID          string
	SuiteID     string
	VersionID   string
	Status      string
	Results     string // JSON
	StartedAt   time.Time
	CompletedAt time.Time
}

type BenchmarkRun struct {
	ID          string
	BenchmarkID string
	VersionID   string
	Results     string // JSON
	CreatedAt   time.Time
}

type Comment struct {
	ID         string
	PromptID   string
	VersionID  string
	LineNumber int
	Content    string
	CreatedAt  time.Time
}

type Chain struct {
	ID          string
	Name        string
	Description string
	ProjectID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ChainWithStepCount struct {
	Chain
	StepCount int
}

type ChainStep struct {
	ID           string
	ChainID      string
	StepOrder    int
	PromptName   string
	InputMapping string // JSON
	OutputKey    string
}

type ChainRun struct {
	ID          string
	ChainID     string
	Status      string
	Inputs      string // JSON
	Results     string // JSON
	FinalOutput string
	StartedAt   time.Time
	CompletedAt time.Time
}
