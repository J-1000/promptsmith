package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Prompt, version, and tag persistence.

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
