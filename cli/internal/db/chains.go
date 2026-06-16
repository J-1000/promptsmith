package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Chain methods

func (db *DB) CreateChain(projectID, name, description string) (*Chain, error) {
	chain := &Chain{
		ID:          NewUUID(),
		Name:        name,
		Description: description,
		ProjectID:   projectID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.Exec(
		`INSERT INTO chains (id, name, description, project_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		chain.ID, chain.Name, chain.Description, chain.ProjectID, chain.CreatedAt, chain.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}
	return chain, nil
}

func (db *DB) GetChainByName(name string) (*Chain, error) {
	var c Chain
	err := db.QueryRow(
		`SELECT id, name, description, project_id, created_at, updated_at FROM chains WHERE name = ?`,
		name,
	).Scan(&c.ID, &c.Name, &c.Description, &c.ProjectID, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) GetChainByID(id string) (*Chain, error) {
	var c Chain
	err := db.QueryRow(
		`SELECT id, name, description, project_id, created_at, updated_at FROM chains WHERE id = ?`,
		id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.ProjectID, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) ListChains() ([]*Chain, error) {
	rows, err := db.Query(`SELECT id, name, description, project_id, created_at, updated_at FROM chains ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []*Chain
	for rows.Next() {
		var c Chain
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.ProjectID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		chains = append(chains, &c)
	}
	return chains, nil
}

func (db *DB) ListChainsWithStepCounts() ([]*ChainWithStepCount, error) {
	rows, err := db.Query(`
		SELECT
			c.id, c.name, c.description, c.project_id, c.created_at, c.updated_at,
			COUNT(cs.id) AS step_count
		FROM chains c
		LEFT JOIN chain_steps cs ON cs.chain_id = c.id
		GROUP BY c.id, c.name, c.description, c.project_id, c.created_at, c.updated_at
		ORDER BY c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []*ChainWithStepCount
	for rows.Next() {
		var c ChainWithStepCount
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.ProjectID, &c.CreatedAt, &c.UpdatedAt, &c.StepCount); err != nil {
			return nil, err
		}
		chains = append(chains, &c)
	}
	return chains, nil
}

func (db *DB) UpdateChain(chainID, name, description string) (*Chain, error) {
	now := time.Now()
	_, err := db.Exec(
		`UPDATE chains SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, now, chainID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update chain: %w", err)
	}
	return db.GetChainByID(chainID)
}

func (db *DB) DeleteChain(chainID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM chain_runs WHERE chain_id = ?", chainID); err != nil {
		return fmt.Errorf("failed to delete chain runs: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM chain_steps WHERE chain_id = ?", chainID); err != nil {
		return fmt.Errorf("failed to delete chain steps: %w", err)
	}
	result, err := tx.Exec("DELETE FROM chains WHERE id = ?", chainID)
	if err != nil {
		return fmt.Errorf("failed to delete chain: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("chain not found")
	}
	return tx.Commit()
}

// Chain Step methods

func (db *DB) CreateChainStep(chainID string, stepOrder int, promptName, inputMapping, outputKey string) (*ChainStep, error) {
	step := &ChainStep{
		ID:           NewUUID(),
		ChainID:      chainID,
		StepOrder:    stepOrder,
		PromptName:   promptName,
		InputMapping: inputMapping,
		OutputKey:    outputKey,
	}

	_, err := db.Exec(
		`INSERT INTO chain_steps (id, chain_id, step_order, prompt_name, input_mapping, output_key)
		VALUES (?, ?, ?, ?, ?, ?)`,
		step.ID, step.ChainID, step.StepOrder, step.PromptName, step.InputMapping, step.OutputKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain step: %w", err)
	}
	return step, nil
}

func (db *DB) ListChainSteps(chainID string) ([]*ChainStep, error) {
	rows, err := db.Query(
		`SELECT id, chain_id, step_order, prompt_name, input_mapping, output_key
		FROM chain_steps WHERE chain_id = ? ORDER BY step_order`,
		chainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*ChainStep
	for rows.Next() {
		var s ChainStep
		if err := rows.Scan(&s.ID, &s.ChainID, &s.StepOrder, &s.PromptName, &s.InputMapping, &s.OutputKey); err != nil {
			return nil, err
		}
		steps = append(steps, &s)
	}
	return steps, nil
}

func (db *DB) ReplaceChainSteps(chainID string, steps []ChainStep) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM chain_steps WHERE chain_id = ?", chainID); err != nil {
		return fmt.Errorf("failed to delete old steps: %w", err)
	}

	for _, s := range steps {
		id := NewUUID()
		if _, err := tx.Exec(
			`INSERT INTO chain_steps (id, chain_id, step_order, prompt_name, input_mapping, output_key)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, chainID, s.StepOrder, s.PromptName, s.InputMapping, s.OutputKey,
		); err != nil {
			return fmt.Errorf("failed to insert step %d: %w", s.StepOrder, err)
		}
	}

	return tx.Commit()
}

// Chain Run methods

func (db *DB) SaveChainRun(chainID, status, inputs, results, finalOutput string) (*ChainRun, error) {
	run := &ChainRun{
		ID:          NewUUID(),
		ChainID:     chainID,
		Status:      status,
		Inputs:      inputs,
		Results:     results,
		FinalOutput: finalOutput,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}

	_, err := db.Exec(
		`INSERT INTO chain_runs (id, chain_id, status, inputs, results, final_output, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.ChainID, run.Status, run.Inputs, run.Results, run.FinalOutput, run.StartedAt, run.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save chain run: %w", err)
	}
	return run, nil
}

func (db *DB) ListChainRuns(chainID string) ([]*ChainRun, error) {
	rows, err := db.Query(
		`SELECT id, chain_id, status, inputs, results, final_output, started_at, completed_at
		FROM chain_runs WHERE chain_id = ? ORDER BY started_at DESC`,
		chainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*ChainRun
	for rows.Next() {
		var r ChainRun
		if err := rows.Scan(&r.ID, &r.ChainID, &r.Status, &r.Inputs, &r.Results, &r.FinalOutput, &r.StartedAt, &r.CompletedAt); err != nil {
			return nil, err
		}
		runs = append(runs, &r)
	}
	return runs, nil
}
