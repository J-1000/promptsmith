package db

import (
	"fmt"
	"time"
)

// Comment methods

func (db *DB) CreateComment(promptID, versionID string, lineNumber int, content string) (*Comment, error) {
	c := &Comment{
		ID:         NewUUID(),
		PromptID:   promptID,
		VersionID:  versionID,
		LineNumber: lineNumber,
		Content:    content,
		CreatedAt:  time.Now(),
	}

	_, err := db.Exec(
		`INSERT INTO comments (id, prompt_id, version_id, line_number, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.PromptID, c.VersionID, c.LineNumber, c.Content, c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}
	return c, nil
}

func (db *DB) ListComments(promptID string) ([]*Comment, error) {
	rows, err := db.Query(
		`SELECT id, prompt_id, version_id, line_number, content, created_at
		FROM comments WHERE prompt_id = ? ORDER BY line_number, created_at`,
		promptID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PromptID, &c.VersionID, &c.LineNumber, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	return comments, nil
}

func (db *DB) DeleteComment(commentID string) error {
	result, err := db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("comment not found")
	}
	return nil
}
