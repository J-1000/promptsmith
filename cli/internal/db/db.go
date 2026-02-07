package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ConfigDir  = ".promptsmith"
	DBFile     = "promptsmith.db"
	ConfigFile = "config.yaml"
)

type DB struct {
	*sql.DB
	projectRoot string
}

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

func NewUUID() string {
	return uuid.New().String()
}

func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, ConfigDir)
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a promptsmith project (or any parent): .promptsmith directory not found")
		}
		dir = parent
	}
}

func Open(projectRoot string) (*DB, error) {
	dbPath := filepath.Join(projectRoot, ConfigDir, DBFile)
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{DB: sqlDB, projectRoot: projectRoot}
	return db, nil
}

func Initialize(projectRoot string) (*DB, error) {
	configDir := filepath.Join(projectRoot, ConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	db, err := Open(projectRoot)
	if err != nil {
		return nil, err
	}

	if err := db.createSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS prompts (
		id TEXT PRIMARY KEY,
		project_id TEXT REFERENCES projects(id),
		name TEXT NOT NULL,
		description TEXT,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS prompt_versions (
		id TEXT PRIMARY KEY,
		prompt_id TEXT REFERENCES prompts(id),
		version TEXT NOT NULL,
		content TEXT NOT NULL,
		variables TEXT,
		metadata TEXT,
		parent_version_id TEXT,
		commit_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT
	);

	CREATE TABLE IF NOT EXISTS tags (
		id TEXT PRIMARY KEY,
		prompt_id TEXT REFERENCES prompts(id),
		version_id TEXT REFERENCES prompt_versions(id),
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS test_suites (
		id TEXT PRIMARY KEY,
		prompt_id TEXT REFERENCES prompts(id),
		name TEXT NOT NULL,
		config TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS test_runs (
		id TEXT PRIMARY KEY,
		suite_id TEXT REFERENCES test_suites(id),
		version_id TEXT REFERENCES prompt_versions(id),
		status TEXT,
		results TEXT,
		started_at DATETIME,
		completed_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS benchmarks (
		id TEXT PRIMARY KEY,
		prompt_id TEXT REFERENCES prompts(id),
		config TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS benchmark_runs (
		id TEXT PRIMARY KEY,
		benchmark_id TEXT REFERENCES benchmarks(id),
		version_id TEXT REFERENCES prompt_versions(id),
		results TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_prompts_project ON prompts(project_id);
	CREATE INDEX IF NOT EXISTS idx_versions_prompt ON prompt_versions(prompt_id);
	CREATE INDEX IF NOT EXISTS idx_tags_prompt ON tags(prompt_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func (db *DB) ProjectRoot() string {
	return db.projectRoot
}

func (db *DB) CreateProject(name string) (*Project, error) {
	project := &Project{
		ID:        NewUUID(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := db.Exec(
		"INSERT INTO projects (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)",
		project.ID, project.Name, project.CreatedAt, project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

func (db *DB) GetProject() (*Project, error) {
	var project Project
	err := db.QueryRow("SELECT id, name, created_at, updated_at FROM projects LIMIT 1").Scan(
		&project.ID, &project.Name, &project.CreatedAt, &project.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (db *DB) CreatePrompt(projectID, name, description, filePath string) (*Prompt, error) {
	prompt := &Prompt{
		ID:          NewUUID(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		FilePath:    filePath,
		CreatedAt:   time.Now(),
	}

	_, err := db.Exec(
		"INSERT INTO prompts (id, project_id, name, description, file_path, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		prompt.ID, prompt.ProjectID, prompt.Name, prompt.Description, prompt.FilePath, prompt.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	return prompt, nil
}

func (db *DB) GetPromptByPath(filePath string) (*Prompt, error) {
	var prompt Prompt
	err := db.QueryRow(
		"SELECT id, project_id, name, description, file_path, created_at FROM prompts WHERE file_path = ?",
		filePath,
	).Scan(&prompt.ID, &prompt.ProjectID, &prompt.Name, &prompt.Description, &prompt.FilePath, &prompt.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

func (db *DB) GetPromptByName(name string) (*Prompt, error) {
	var prompt Prompt
	err := db.QueryRow(
		"SELECT id, project_id, name, description, file_path, created_at FROM prompts WHERE name = ?",
		name,
	).Scan(&prompt.ID, &prompt.ProjectID, &prompt.Name, &prompt.Description, &prompt.FilePath, &prompt.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

func (db *DB) ListPrompts() ([]*Prompt, error) {
	rows, err := db.Query("SELECT id, project_id, name, description, file_path, created_at FROM prompts ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []*Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.FilePath, &p.CreatedAt); err != nil {
			return nil, err
		}
		prompts = append(prompts, &p)
	}
	return prompts, nil
}

func (db *DB) CreateVersion(promptID, version, content, variables, metadata, commitMessage, createdBy string, parentVersionID *string) (*PromptVersion, error) {
	v := &PromptVersion{
		ID:              NewUUID(),
		PromptID:        promptID,
		Version:         version,
		Content:         content,
		Variables:       variables,
		Metadata:        metadata,
		ParentVersionID: parentVersionID,
		CommitMessage:   commitMessage,
		CreatedAt:       time.Now(),
		CreatedBy:       createdBy,
	}

	_, err := db.Exec(
		`INSERT INTO prompt_versions
		(id, prompt_id, version, content, variables, metadata, parent_version_id, commit_message, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.PromptID, v.Version, v.Content, v.Variables, v.Metadata, v.ParentVersionID, v.CommitMessage, v.CreatedAt, v.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return v, nil
}

func (db *DB) GetLatestVersion(promptID string) (*PromptVersion, error) {
	var v PromptVersion
	var parentID sql.NullString
	err := db.QueryRow(
		`SELECT id, prompt_id, version, content, variables, metadata, parent_version_id, commit_message, created_at, created_by
		FROM prompt_versions WHERE prompt_id = ? ORDER BY created_at DESC LIMIT 1`,
		promptID,
	).Scan(&v.ID, &v.PromptID, &v.Version, &v.Content, &v.Variables, &v.Metadata, &parentID, &v.CommitMessage, &v.CreatedAt, &v.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		v.ParentVersionID = &parentID.String
	}
	return &v, nil
}

func (db *DB) ListVersions(promptID string) ([]*PromptVersion, error) {
	rows, err := db.Query(
		`SELECT id, prompt_id, version, content, variables, metadata, parent_version_id, commit_message, created_at, created_by
		FROM prompt_versions WHERE prompt_id = ? ORDER BY created_at DESC`,
		promptID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*PromptVersion
	for rows.Next() {
		var v PromptVersion
		var parentID sql.NullString
		if err := rows.Scan(&v.ID, &v.PromptID, &v.Version, &v.Content, &v.Variables, &v.Metadata, &parentID, &v.CommitMessage, &v.CreatedAt, &v.CreatedBy); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v.ParentVersionID = &parentID.String
		}
		versions = append(versions, &v)
	}
	return versions, nil
}

func (db *DB) GetVersionByString(promptID, version string) (*PromptVersion, error) {
	var v PromptVersion
	var parentID sql.NullString
	err := db.QueryRow(
		`SELECT id, prompt_id, version, content, variables, metadata, parent_version_id, commit_message, created_at, created_by
		FROM prompt_versions WHERE prompt_id = ? AND version = ?`,
		promptID, version,
	).Scan(&v.ID, &v.PromptID, &v.Version, &v.Content, &v.Variables, &v.Metadata, &parentID, &v.CommitMessage, &v.CreatedAt, &v.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		v.ParentVersionID = &parentID.String
	}
	return &v, nil
}

func (db *DB) GetVersionByID(id string) (*PromptVersion, error) {
	var v PromptVersion
	var parentID sql.NullString
	err := db.QueryRow(
		`SELECT id, prompt_id, version, content, variables, metadata, parent_version_id, commit_message, created_at, created_by
		FROM prompt_versions WHERE id = ?`,
		id,
	).Scan(&v.ID, &v.PromptID, &v.Version, &v.Content, &v.Variables, &v.Metadata, &parentID, &v.CommitMessage, &v.CreatedAt, &v.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		v.ParentVersionID = &parentID.String
	}
	return &v, nil
}

func (db *DB) CreateTag(promptID, versionID, name string) (*Tag, error) {
	// Check if tag already exists
	existing, err := db.GetTagByName(promptID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update existing tag to point to new version
		_, err := db.Exec("UPDATE tags SET version_id = ? WHERE id = ?", versionID, existing.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to update tag: %w", err)
		}
		existing.VersionID = versionID
		return existing, nil
	}

	tag := &Tag{
		ID:        NewUUID(),
		PromptID:  promptID,
		VersionID: versionID,
		Name:      name,
		CreatedAt: time.Now(),
	}

	_, err = db.Exec(
		"INSERT INTO tags (id, prompt_id, version_id, name, created_at) VALUES (?, ?, ?, ?, ?)",
		tag.ID, tag.PromptID, tag.VersionID, tag.Name, tag.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return tag, nil
}

func (db *DB) GetTagByName(promptID, name string) (*Tag, error) {
	var tag Tag
	err := db.QueryRow(
		"SELECT id, prompt_id, version_id, name, created_at FROM tags WHERE prompt_id = ? AND name = ?",
		promptID, name,
	).Scan(&tag.ID, &tag.PromptID, &tag.VersionID, &tag.Name, &tag.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (db *DB) ListTags(promptID string) ([]*Tag, error) {
	rows, err := db.Query(
		"SELECT id, prompt_id, version_id, name, created_at FROM tags WHERE prompt_id = ? ORDER BY name",
		promptID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.PromptID, &t.VersionID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, &t)
	}
	return tags, nil
}

func (db *DB) DeleteTag(promptID, name string) error {
	result, err := db.Exec("DELETE FROM tags WHERE prompt_id = ? AND name = ?", promptID, name)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tag '%s' not found", name)
	}
	return nil
}

func (db *DB) UpdatePrompt(promptID, name, description string) (*Prompt, error) {
	_, err := db.Exec(
		"UPDATE prompts SET name = ?, description = ? WHERE id = ?",
		name, description, promptID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update prompt: %w", err)
	}

	var p Prompt
	err = db.QueryRow(
		"SELECT id, project_id, name, description, file_path, created_at FROM prompts WHERE id = ?",
		promptID,
	).Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.FilePath, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) DeletePrompt(promptID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete tags first (references versions and prompts)
	if _, err := tx.Exec("DELETE FROM tags WHERE prompt_id = ?", promptID); err != nil {
		return fmt.Errorf("failed to delete tags: %w", err)
	}

	// Delete versions (references prompts)
	if _, err := tx.Exec("DELETE FROM prompt_versions WHERE prompt_id = ?", promptID); err != nil {
		return fmt.Errorf("failed to delete versions: %w", err)
	}

	// Delete the prompt itself
	result, err := tx.Exec("DELETE FROM prompts WHERE id = ?", promptID)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("prompt not found")
	}

	return tx.Commit()
}

func (db *DB) GetAllVersionsForLog() ([]struct {
	Prompt  *Prompt
	Version *PromptVersion
}, error) {
	rows, err := db.Query(`
		SELECT p.id, p.project_id, p.name, p.description, p.file_path, p.created_at,
			   v.id, v.prompt_id, v.version, v.content, v.variables, v.metadata, v.parent_version_id, v.commit_message, v.created_at, v.created_by
		FROM prompt_versions v
		JOIN prompts p ON v.prompt_id = p.id
		ORDER BY v.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		Prompt  *Prompt
		Version *PromptVersion
	}
	for rows.Next() {
		var p Prompt
		var v PromptVersion
		var parentID sql.NullString
		if err := rows.Scan(
			&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.FilePath, &p.CreatedAt,
			&v.ID, &v.PromptID, &v.Version, &v.Content, &v.Variables, &v.Metadata, &parentID, &v.CommitMessage, &v.CreatedAt, &v.CreatedBy,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v.ParentVersionID = &parentID.String
		}
		results = append(results, struct {
			Prompt  *Prompt
			Version *PromptVersion
		}{&p, &v})
	}
	return results, nil
}

// Test Run methods

func (db *DB) SaveTestRun(suiteID, versionID, status, results string) (*TestRun, error) {
	run := &TestRun{
		ID:          NewUUID(),
		SuiteID:     suiteID,
		VersionID:   versionID,
		Status:      status,
		Results:     results,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}

	_, err := db.Exec(
		`INSERT INTO test_runs (id, suite_id, version_id, status, results, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.SuiteID, run.VersionID, run.Status, run.Results, run.StartedAt, run.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save test run: %w", err)
	}
	return run, nil
}

func (db *DB) ListTestRuns(suiteID string) ([]*TestRun, error) {
	rows, err := db.Query(
		`SELECT id, suite_id, version_id, status, results, started_at, completed_at
		FROM test_runs WHERE suite_id = ? ORDER BY started_at DESC`,
		suiteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*TestRun
	for rows.Next() {
		var r TestRun
		if err := rows.Scan(&r.ID, &r.SuiteID, &r.VersionID, &r.Status, &r.Results, &r.StartedAt, &r.CompletedAt); err != nil {
			return nil, err
		}
		runs = append(runs, &r)
	}
	return runs, nil
}

func (db *DB) GetTestRun(runID string) (*TestRun, error) {
	var r TestRun
	err := db.QueryRow(
		`SELECT id, suite_id, version_id, status, results, started_at, completed_at
		FROM test_runs WHERE id = ?`,
		runID,
	).Scan(&r.ID, &r.SuiteID, &r.VersionID, &r.Status, &r.Results, &r.StartedAt, &r.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Benchmark Run methods

func (db *DB) SaveBenchmarkRun(benchmarkID, versionID, results string) (*BenchmarkRun, error) {
	run := &BenchmarkRun{
		ID:          NewUUID(),
		BenchmarkID: benchmarkID,
		VersionID:   versionID,
		Results:     results,
		CreatedAt:   time.Now(),
	}

	_, err := db.Exec(
		`INSERT INTO benchmark_runs (id, benchmark_id, version_id, results, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		run.ID, run.BenchmarkID, run.VersionID, run.Results, run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save benchmark run: %w", err)
	}
	return run, nil
}

func (db *DB) ListBenchmarkRuns(benchmarkID string) ([]*BenchmarkRun, error) {
	rows, err := db.Query(
		`SELECT id, benchmark_id, version_id, results, created_at
		FROM benchmark_runs WHERE benchmark_id = ? ORDER BY created_at DESC`,
		benchmarkID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*BenchmarkRun
	for rows.Next() {
		var r BenchmarkRun
		if err := rows.Scan(&r.ID, &r.BenchmarkID, &r.VersionID, &r.Results, &r.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, &r)
	}
	return runs, nil
}
