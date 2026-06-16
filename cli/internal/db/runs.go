package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Test suite, benchmark, and run persistence.

func (db *DB) EnsureTestSuite(id, promptID, name, config string) error {
	if id == "" {
		return fmt.Errorf("test suite id is required")
	}
	if config == "" {
		config = "{}"
	}

	_, err := db.Exec(
		`INSERT INTO test_suites (id, prompt_id, name, config)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET prompt_id = excluded.prompt_id, name = excluded.name, config = excluded.config`,
		id, promptID, name, config,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert test suite: %w", err)
	}
	return nil
}

func (db *DB) EnsureBenchmark(id, promptID, config string) error {
	if id == "" {
		return fmt.Errorf("benchmark id is required")
	}
	if config == "" {
		config = "{}"
	}

	_, err := db.Exec(
		`INSERT INTO benchmarks (id, prompt_id, config)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET prompt_id = excluded.prompt_id, config = excluded.config`,
		id, promptID, config,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert benchmark: %w", err)
	}
	return nil
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
	versionValue := nullIfEmpty(versionID)
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
		run.ID, run.SuiteID, versionValue, run.Status, run.Results, run.StartedAt, run.CompletedAt,
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
		var versionID sql.NullString
		if err := rows.Scan(&r.ID, &r.SuiteID, &versionID, &r.Status, &r.Results, &r.StartedAt, &r.CompletedAt); err != nil {
			return nil, err
		}
		r.VersionID = stringFromNull(versionID)
		runs = append(runs, &r)
	}
	return runs, nil
}

func (db *DB) GetTestRun(runID string) (*TestRun, error) {
	var r TestRun
	row := db.QueryRow(
		`SELECT id, suite_id, version_id, status, results, started_at, completed_at
		FROM test_runs WHERE id = ?`,
		runID,
	)
	var versionID sql.NullString
	err := row.Scan(&r.ID, &r.SuiteID, &versionID, &r.Status, &r.Results, &r.StartedAt, &r.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.VersionID = stringFromNull(versionID)
	return &r, nil
}

// Benchmark Run methods

func (db *DB) SaveBenchmarkRun(benchmarkID, versionID, results string) (*BenchmarkRun, error) {
	versionValue := nullIfEmpty(versionID)
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
		run.ID, run.BenchmarkID, versionValue, run.Results, run.CreatedAt,
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
		var versionID sql.NullString
		if err := rows.Scan(&r.ID, &r.BenchmarkID, &versionID, &r.Results, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.VersionID = stringFromNull(versionID)
		runs = append(runs, &r)
	}
	return runs, nil
}
