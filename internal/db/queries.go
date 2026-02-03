package db

import (
	"database/sql"
	"fmt"
	"time"
)

func (d *DB) CreateSession(s *Session) error {
	query := `
		INSERT INTO sessions (id, title, provider, model, created_at, updated_at, archived)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := d.db.Exec(query,
		s.ID,
		s.Title,
		s.Provider,
		s.Model,
		s.CreatedAt.Format(time.RFC3339),
		s.UpdatedAt.Format(time.RFC3339),
		s.Archived,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (d *DB) GetSession(id string) (*Session, error) {
	query := `
		SELECT id, title, provider, model, created_at, updated_at, archived
		FROM sessions
		WHERE id = ?
	`
	var s Session
	var createdAt, updatedAt string
	var archived int

	err := d.db.QueryRow(query, id).Scan(
		&s.ID,
		&s.Title,
		&s.Provider,
		&s.Model,
		&createdAt,
		&updatedAt,
		&archived,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	s.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	s.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	s.Archived = archived != 0

	return &s, nil
}

func (d *DB) ListSessions(includeArchived bool) ([]Session, error) {
	query := `
		SELECT id, title, provider, model, created_at, updated_at, archived
		FROM sessions
	`
	if !includeArchived {
		query += " WHERE archived = 0"
	}
	query += " ORDER BY created_at DESC"

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var createdAt, updatedAt string
		var archived int

		if err := rows.Scan(
			&s.ID,
			&s.Title,
			&s.Provider,
			&s.Model,
			&createdAt,
			&updatedAt,
			&archived,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		s.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		s.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}

		s.Archived = archived != 0

		sessions = append(sessions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

func (d *DB) UpdateSessionTitle(id, title string) error {
	query := `
		UPDATE sessions
		SET title = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := d.db.Exec(query, title, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update session title: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}

func (d *DB) ArchiveSession(id string) error {
	query := `
		UPDATE sessions
		SET archived = 1, updated_at = ?
		WHERE id = ?
	`
	result, err := d.db.Exec(query, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to archive session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}

func (d *DB) UnarchiveSession(id string) error {
	query := `
		UPDATE sessions
		SET archived = 0, updated_at = ?
		WHERE id = ?
	`
	result, err := d.db.Exec(query, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to unarchive session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}

func (d *DB) AddMessage(m *Message) error {
	query := `
		INSERT INTO messages (session_id, role, content, created_at, tokens)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := d.db.Exec(query,
		m.SessionID,
		m.Role,
		m.Content,
		m.CreatedAt.Format(time.RFC3339),
		m.Tokens,
	)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	m.ID = id
	return nil
}

func (d *DB) GetSessionMessages(sessionID string) ([]Message, error) {
	query := `
		SELECT id, session_id, role, content, created_at, tokens
		FROM messages
		WHERE session_id = ?
		ORDER BY created_at ASC
	`
	rows, err := d.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var createdAt string

		if err := rows.Scan(
			&m.ID,
			&m.SessionID,
			&m.Role,
			&m.Content,
			&createdAt,
			&m.Tokens,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		m.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

func (d *DB) DeleteSession(id string) error {
	// Delete messages first (foreign key constraint)
	_, err := d.db.Exec("DELETE FROM messages WHERE session_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete session messages: %w", err)
	}

	// Delete session
	result, err := d.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}
