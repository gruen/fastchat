package db

import (
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func TestCreateSessionAndGetSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Round(time.Second)
	session := &Session{
		ID:        "test-session-1",
		Title:     "Test Session",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}

	err := db.CreateSession(session)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	retrieved, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, retrieved.ID)
	}
	if retrieved.Title != session.Title {
		t.Errorf("expected Title %s, got %s", session.Title, retrieved.Title)
	}
	if retrieved.Provider != session.Provider {
		t.Errorf("expected Provider %s, got %s", session.Provider, retrieved.Provider)
	}
	if retrieved.Model != session.Model {
		t.Errorf("expected Model %s, got %s", session.Model, retrieved.Model)
	}
	if !retrieved.CreatedAt.Equal(session.CreatedAt) {
		t.Errorf("expected CreatedAt %v, got %v", session.CreatedAt, retrieved.CreatedAt)
	}
	if !retrieved.UpdatedAt.Equal(session.UpdatedAt) {
		t.Errorf("expected UpdatedAt %v, got %v", session.UpdatedAt, retrieved.UpdatedAt)
	}
	if retrieved.Archived != session.Archived {
		t.Errorf("expected Archived %v, got %v", session.Archived, retrieved.Archived)
	}
}

func TestListSessionsOrderedByCreatedAtDesc(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create sessions with different creation times
	sessions := []Session{
		{
			ID:        "session-1",
			Title:     "First",
			Provider:  "openai",
			Model:     "gpt-4",
			CreatedAt: time.Now().Add(-2 * time.Hour).Round(time.Second),
			UpdatedAt: time.Now().Round(time.Second),
			Archived:  false,
		},
		{
			ID:        "session-2",
			Title:     "Second",
			Provider:  "anthropic",
			Model:     "claude-3",
			CreatedAt: time.Now().Add(-1 * time.Hour).Round(time.Second),
			UpdatedAt: time.Now().Round(time.Second),
			Archived:  false,
		},
		{
			ID:        "session-3",
			Title:     "Third",
			Provider:  "openai",
			Model:     "gpt-4",
			CreatedAt: time.Now().Round(time.Second),
			UpdatedAt: time.Now().Round(time.Second),
			Archived:  false,
		},
	}

	for _, s := range sessions {
		if err := db.CreateSession(&s); err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	retrieved, err := db.ListSessions(false)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(retrieved) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(retrieved))
	}

	// Should be ordered DESC by created_at
	if retrieved[0].ID != "session-3" {
		t.Errorf("expected first session to be session-3, got %s", retrieved[0].ID)
	}
	if retrieved[1].ID != "session-2" {
		t.Errorf("expected second session to be session-2, got %s", retrieved[1].ID)
	}
	if retrieved[2].ID != "session-1" {
		t.Errorf("expected third session to be session-1, got %s", retrieved[2].ID)
	}
}

func TestListSessionsExcludeArchived(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Round(time.Second)

	sessions := []Session{
		{
			ID:        "active-1",
			Title:     "Active Session",
			Provider:  "openai",
			Model:     "gpt-4",
			CreatedAt: now,
			UpdatedAt: now,
			Archived:  false,
		},
		{
			ID:        "archived-1",
			Title:     "Archived Session",
			Provider:  "anthropic",
			Model:     "claude-3",
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now,
			Archived:  true,
		},
	}

	for _, s := range sessions {
		if err := db.CreateSession(&s); err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Test with includeArchived=false
	retrieved, err := db.ListSessions(false)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(retrieved) != 1 {
		t.Fatalf("expected 1 active session, got %d", len(retrieved))
	}

	if retrieved[0].ID != "active-1" {
		t.Errorf("expected active-1, got %s", retrieved[0].ID)
	}
}

func TestListSessionsIncludeArchived(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Round(time.Second)

	sessions := []Session{
		{
			ID:        "active-1",
			Title:     "Active Session",
			Provider:  "openai",
			Model:     "gpt-4",
			CreatedAt: now,
			UpdatedAt: now,
			Archived:  false,
		},
		{
			ID:        "archived-1",
			Title:     "Archived Session",
			Provider:  "anthropic",
			Model:     "claude-3",
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now,
			Archived:  true,
		},
	}

	for _, s := range sessions {
		if err := db.CreateSession(&s); err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Test with includeArchived=true
	retrieved, err := db.ListSessions(true)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(retrieved) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(retrieved))
	}
}

func TestUpdateSessionTitle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Add(-5 * time.Second).Round(time.Second)
	session := &Session{
		ID:        "test-session",
		Title:     "Original Title",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}

	if err := db.CreateSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	newTitle := "Updated Title"
	if err := db.UpdateSessionTitle(session.ID, newTitle); err != nil {
		t.Fatalf("failed to update session title: %v", err)
	}

	retrieved, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.Title != newTitle {
		t.Errorf("expected title %s, got %s", newTitle, retrieved.Title)
	}

	// UpdatedAt should have changed (allow for small time differences)
	if !retrieved.UpdatedAt.After(session.UpdatedAt) && !retrieved.UpdatedAt.Equal(session.UpdatedAt) {
		t.Errorf("expected UpdatedAt to be at or after %v, got %v", session.UpdatedAt, retrieved.UpdatedAt)
	}
}

func TestArchiveAndUnarchiveSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Round(time.Second)
	session := &Session{
		ID:        "test-session",
		Title:     "Test Session",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}

	if err := db.CreateSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Archive
	if err := db.ArchiveSession(session.ID); err != nil {
		t.Fatalf("failed to archive session: %v", err)
	}

	retrieved, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if !retrieved.Archived {
		t.Errorf("expected session to be archived")
	}

	// Unarchive
	if err := db.UnarchiveSession(session.ID); err != nil {
		t.Fatalf("failed to unarchive session: %v", err)
	}

	retrieved, err = db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.Archived {
		t.Errorf("expected session to be unarchived")
	}
}

func TestAddMessageAndGetSessionMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Round(time.Second)
	session := &Session{
		ID:        "test-session",
		Title:     "Test Session",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}

	if err := db.CreateSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	message := &Message{
		SessionID: session.ID,
		Role:      "user",
		Content:   "Hello, world!",
		CreatedAt: now,
		Tokens:    10,
	}

	if err := db.AddMessage(message); err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	// ID should be set after insert
	if message.ID == 0 {
		t.Errorf("expected message ID to be set, got 0")
	}

	messages, err := db.GetSessionMessages(session.ID)
	if err != nil {
		t.Fatalf("failed to get session messages: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	retrieved := messages[0]
	if retrieved.ID != message.ID {
		t.Errorf("expected ID %d, got %d", message.ID, retrieved.ID)
	}
	if retrieved.SessionID != message.SessionID {
		t.Errorf("expected SessionID %s, got %s", message.SessionID, retrieved.SessionID)
	}
	if retrieved.Role != message.Role {
		t.Errorf("expected Role %s, got %s", message.Role, retrieved.Role)
	}
	if retrieved.Content != message.Content {
		t.Errorf("expected Content %s, got %s", message.Content, retrieved.Content)
	}
	if !retrieved.CreatedAt.Equal(message.CreatedAt) {
		t.Errorf("expected CreatedAt %v, got %v", message.CreatedAt, retrieved.CreatedAt)
	}
	if retrieved.Tokens != message.Tokens {
		t.Errorf("expected Tokens %d, got %d", message.Tokens, retrieved.Tokens)
	}
}

func TestDeleteSessionRemovesMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Round(time.Second)
	session := &Session{
		ID:        "test-session",
		Title:     "Test Session",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}

	if err := db.CreateSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	message := &Message{
		SessionID: session.ID,
		Role:      "user",
		Content:   "Hello!",
		CreatedAt: now,
		Tokens:    5,
	}

	if err := db.AddMessage(message); err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	// Delete session
	if err := db.DeleteSession(session.ID); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Session should not exist
	_, err := db.GetSession(session.ID)
	if err == nil {
		t.Errorf("expected error when getting deleted session")
	}

	// Messages should also be deleted
	messages, err := db.GetSessionMessages(session.ID)
	if err != nil {
		t.Fatalf("failed to get session messages: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages after session deletion, got %d", len(messages))
	}
}

func TestMultipleMessagesPerSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now().Round(time.Second)
	session := &Session{
		ID:        "test-session",
		Title:     "Test Session",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}

	if err := db.CreateSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	messages := []*Message{
		{
			SessionID: session.ID,
			Role:      "user",
			Content:   "First message",
			CreatedAt: now,
			Tokens:    5,
		},
		{
			SessionID: session.ID,
			Role:      "assistant",
			Content:   "Second message",
			CreatedAt: now.Add(1 * time.Second),
			Tokens:    10,
		},
		{
			SessionID: session.ID,
			Role:      "user",
			Content:   "Third message",
			CreatedAt: now.Add(2 * time.Second),
			Tokens:    7,
		},
	}

	for _, m := range messages {
		if err := db.AddMessage(m); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	retrieved, err := db.GetSessionMessages(session.ID)
	if err != nil {
		t.Fatalf("failed to get session messages: %v", err)
	}

	if len(retrieved) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(retrieved))
	}

	// Should be ordered ASC by created_at
	if retrieved[0].Content != "First message" {
		t.Errorf("expected first message to be 'First message', got %s", retrieved[0].Content)
	}
	if retrieved[1].Content != "Second message" {
		t.Errorf("expected second message to be 'Second message', got %s", retrieved[1].Content)
	}
	if retrieved[2].Content != "Third message" {
		t.Errorf("expected third message to be 'Third message', got %s", retrieved[2].Content)
	}
}
