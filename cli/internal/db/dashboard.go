package db

import (
	"fmt"
	"time"
)

// Dashboard methods

type ActivityEvent struct {
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	Timestamp  time.Time `json:"timestamp"`
	PromptName string    `json:"prompt_name"`
}

func (db *DB) GetRecentActivity(limit int) ([]ActivityEvent, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT type, title, detail, timestamp, prompt_name FROM (
			SELECT 'version' AS type,
				'v' || pv.version AS title,
				pv.commit_message AS detail,
				pv.created_at AS timestamp,
				p.name AS prompt_name
			FROM prompt_versions pv
			JOIN prompts p ON pv.prompt_id = p.id

			UNION ALL

			SELECT 'test_run' AS type,
				tr.status AS title,
				tr.suite_id AS detail,
				tr.completed_at AS timestamp,
				COALESCE(ts.name, tr.suite_id) AS prompt_name
			FROM test_runs tr
			LEFT JOIN test_suites ts ON tr.suite_id = ts.id

			UNION ALL

			SELECT 'benchmark_run' AS type,
				'completed' AS title,
				br.benchmark_id AS detail,
				br.created_at AS timestamp,
				COALESCE(b.id, br.benchmark_id) AS prompt_name
			FROM benchmark_runs br
			LEFT JOIN benchmarks b ON br.benchmark_id = b.id
		) activity
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query activity: %w", err)
	}
	defer rows.Close()

	var events []ActivityEvent
	for rows.Next() {
		var e ActivityEvent
		if err := rows.Scan(&e.Type, &e.Title, &e.Detail, &e.Timestamp, &e.PromptName); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

type PromptHealth struct {
	PromptName     string  `json:"prompt_name"`
	VersionCount   int     `json:"version_count"`
	LastTestStatus string  `json:"last_test_status"`
	LastTestAt     string  `json:"last_test_at"`
	TestPassRate   float64 `json:"test_pass_rate"`
}

func (db *DB) GetPromptHealth() ([]PromptHealth, error) {
	query := `
		SELECT
			p.name,
			(SELECT COUNT(*) FROM prompt_versions pv WHERE pv.prompt_id = p.id) AS version_count,
			COALESCE(
				(SELECT tr.status FROM test_runs tr
				 JOIN test_suites ts ON tr.suite_id = ts.id
				 WHERE ts.prompt_id = p.id
				 ORDER BY tr.completed_at DESC LIMIT 1),
				'none'
			) AS last_test_status,
			COALESCE(
				(SELECT tr.completed_at FROM test_runs tr
				 JOIN test_suites ts ON tr.suite_id = ts.id
				 WHERE ts.prompt_id = p.id
				 ORDER BY tr.completed_at DESC LIMIT 1),
				''
			) AS last_test_at,
			COALESCE(
				(SELECT CAST(SUM(CASE WHEN tr2.status = 'passed' THEN 1 ELSE 0 END) AS REAL) / COUNT(*)
				 FROM test_runs tr2
				 JOIN test_suites ts2 ON tr2.suite_id = ts2.id
				 WHERE ts2.prompt_id = p.id),
				0.0
			) AS test_pass_rate
		FROM prompts p
		ORDER BY p.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query prompt health: %w", err)
	}
	defer rows.Close()

	var results []PromptHealth
	for rows.Next() {
		var h PromptHealth
		if err := rows.Scan(&h.PromptName, &h.VersionCount, &h.LastTestStatus, &h.LastTestAt, &h.TestPassRate); err != nil {
			return nil, err
		}
		results = append(results, h)
	}
	return results, nil
}
