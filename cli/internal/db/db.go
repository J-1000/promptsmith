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

// maxOpenConns bounds the connection pool. WAL mode permits concurrent
// readers alongside a single writer, so the API server can serve overlapping
// requests instead of serializing every query behind one connection. Write
// contention is absorbed by the busy_timeout pragma rather than surfacing as
// "database is locked" errors.
const maxOpenConns = 8

func Open(projectRoot string) (*DB, error) {
	dbPath := filepath.Join(projectRoot, ConfigDir, DBFile)
	// Pragmas are encoded in the DSN so they apply to every connection in the
	// pool. Executing PRAGMA on the *sql.DB handle would only configure a
	// single connection, leaving the rest of the pool with default settings.
	dsn := dbPath + "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on"
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{DB: sqlDB, projectRoot: projectRoot}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func Initialize(projectRoot string) (*DB, error) {
	configDir := filepath.Join(projectRoot, ConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Open runs migrations, which create the schema on a fresh database.
	return Open(projectRoot)
}

// migrations is the ordered list of schema migrations. Each entry advances the
// database by one version; the applied version is tracked in SQLite's
// PRAGMA user_version. Append new migrations to the end — never edit or reorder
// existing entries, as that would corrupt already-migrated databases.
var migrations = []string{
	schemaV1,
}

// migrate applies any migrations newer than the database's current
// user_version, each within its own transaction.
func (db *DB) migrate() error {
	var current int
	if err := db.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	for v := current; v < len(migrations); v++ {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin migration %d: %w", v+1, err)
		}
		if _, err := tx.Exec(migrations[v]); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", v+1, err)
		}
		// user_version is a pragma and cannot be parameterized; v+1 is a
		// trusted loop counter, so the formatting is safe.
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", v+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", v+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", v+1, err)
		}
	}
	return nil
}

const schemaV1 = `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS prompts (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		description TEXT,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(project_id, name),
		UNIQUE(project_id, file_path)
	);

	CREATE TABLE IF NOT EXISTS prompt_versions (
		id TEXT PRIMARY KEY,
		prompt_id TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		version TEXT NOT NULL,
		content TEXT NOT NULL,
		variables TEXT,
		metadata TEXT,
		parent_version_id TEXT REFERENCES prompt_versions(id) ON DELETE SET NULL,
		commit_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		UNIQUE(prompt_id, version)
	);

	CREATE TABLE IF NOT EXISTS tags (
		id TEXT PRIMARY KEY,
		prompt_id TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		version_id TEXT NOT NULL REFERENCES prompt_versions(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(prompt_id, name)
	);

	CREATE TABLE IF NOT EXISTS test_suites (
		id TEXT PRIMARY KEY,
		prompt_id TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		config TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS test_runs (
		id TEXT PRIMARY KEY,
		suite_id TEXT NOT NULL REFERENCES test_suites(id) ON DELETE CASCADE,
		version_id TEXT REFERENCES prompt_versions(id) ON DELETE SET NULL,
		status TEXT,
		results TEXT,
		started_at DATETIME,
		completed_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS benchmarks (
		id TEXT PRIMARY KEY,
		prompt_id TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		config TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS benchmark_runs (
		id TEXT PRIMARY KEY,
		benchmark_id TEXT NOT NULL REFERENCES benchmarks(id) ON DELETE CASCADE,
		version_id TEXT REFERENCES prompt_versions(id) ON DELETE SET NULL,
		results TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS comments (
		id TEXT PRIMARY KEY,
		prompt_id TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		version_id TEXT NOT NULL REFERENCES prompt_versions(id) ON DELETE CASCADE,
		line_number INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chains (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chain_steps (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
		step_order INTEGER NOT NULL,
		prompt_name TEXT NOT NULL,
		input_mapping TEXT,
		output_key TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS chain_runs (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
		status TEXT,
		inputs TEXT,
		results TEXT,
		final_output TEXT,
		started_at DATETIME,
		completed_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_prompts_project ON prompts(project_id);
	CREATE INDEX IF NOT EXISTS idx_versions_prompt ON prompt_versions(prompt_id);
	CREATE INDEX IF NOT EXISTS idx_tags_prompt ON tags(prompt_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_prompts_project_name_unique ON prompts(project_id, name);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_prompts_project_path_unique ON prompts(project_id, file_path);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_versions_prompt_version_unique ON prompt_versions(prompt_id, version);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_prompt_name_unique ON tags(prompt_id, name);
	CREATE INDEX IF NOT EXISTS idx_comments_prompt ON comments(prompt_id);
	CREATE INDEX IF NOT EXISTS idx_test_suites_prompt ON test_suites(prompt_id);
	CREATE INDEX IF NOT EXISTS idx_test_runs_suite ON test_runs(suite_id);
	CREATE INDEX IF NOT EXISTS idx_benchmarks_prompt ON benchmarks(prompt_id);
	CREATE INDEX IF NOT EXISTS idx_benchmark_runs_benchmark ON benchmark_runs(benchmark_id);
	CREATE INDEX IF NOT EXISTS idx_chains_project ON chains(project_id);
	CREATE INDEX IF NOT EXISTS idx_chain_steps_chain ON chain_steps(chain_id);
	CREATE INDEX IF NOT EXISTS idx_chain_runs_chain ON chain_runs(chain_id);
	`

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

func (db *DB) ListPromptsWithLatestVersion() ([]*PromptWithLatestVersion, error) {
	rows, err := db.Query(`
		SELECT
			p.id, p.project_id, p.name, p.description, p.file_path, p.created_at,
			(
				SELECT pv.version
				FROM prompt_versions pv
				WHERE pv.prompt_id = p.id
				ORDER BY pv.created_at DESC
				LIMIT 1
			) AS latest_version
		FROM prompts p
		ORDER BY p.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []*PromptWithLatestVersion
	for rows.Next() {
		var p PromptWithLatestVersion
		var latestVersion sql.NullString
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.FilePath, &p.CreatedAt, &latestVersion); err != nil {
			return nil, err
		}
		if latestVersion.Valid {
			p.LatestVersion = latestVersion.String
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
	version, err := db.GetVersionByID(versionID)
	if err != nil {
		return nil, err
	}
	if version == nil || version.PromptID != promptID {
		return nil, fmt.Errorf("version does not belong to prompt")
	}

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
	var promptName string
	var projectID string
	err := db.QueryRow("SELECT name, project_id FROM prompts WHERE id = ?", promptID).Scan(&promptName, &projectID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("prompt not found")
	}
	if err != nil {
		return fmt.Errorf("failed to find prompt: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM comments WHERE prompt_id = ?", promptID); err != nil {
		return fmt.Errorf("failed to delete comments: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tags WHERE prompt_id = ?", promptID); err != nil {
		return fmt.Errorf("failed to delete tags: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM test_runs WHERE suite_id IN (SELECT id FROM test_suites WHERE prompt_id = ?)", promptID); err != nil {
		return fmt.Errorf("failed to delete test runs: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM test_suites WHERE prompt_id = ?", promptID); err != nil {
		return fmt.Errorf("failed to delete test suites: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM benchmark_runs WHERE benchmark_id IN (SELECT id FROM benchmarks WHERE prompt_id = ?)", promptID); err != nil {
		return fmt.Errorf("failed to delete benchmark runs: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM benchmarks WHERE prompt_id = ?", promptID); err != nil {
		return fmt.Errorf("failed to delete benchmarks: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM chain_steps WHERE prompt_name = ? AND chain_id IN (SELECT id FROM chains WHERE project_id = ?)", promptName, projectID); err != nil {
		return fmt.Errorf("failed to delete chain steps: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM prompt_versions WHERE prompt_id = ?", promptID); err != nil {
		return fmt.Errorf("failed to delete versions: %w", err)
	}

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

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func stringFromNull(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}
