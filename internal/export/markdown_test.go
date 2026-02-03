package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mg/ai-tui/internal/db"
)

func TestToMarkdown_Format(t *testing.T) {
	dir := t.TempDir()

	createdAt := time.Date(2026, 2, 3, 14, 30, 0, 0, time.UTC)

	session := db.Session{
		ID:        "session-123",
		Title:     "Test Session",
		Provider:  "anthropic",
		Model:     "claude-opus-4",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
	}

	messages := []db.Message{
		{
			ID:        1,
			SessionID: "session-123",
			Role:      "user",
			Content:   "Hello, how are you?",
			CreatedAt: createdAt,
			Tokens:    5,
		},
		{
			ID:        2,
			SessionID: "session-123",
			Role:      "assistant",
			Content:   "I'm doing well, thank you!",
			CreatedAt: createdAt.Add(time.Second),
			Tokens:    7,
		},
	}

	filePath, err := ToMarkdown(session, messages, dir)
	if err != nil {
		t.Fatalf("ToMarkdown failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist at %s", filePath)
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Verify expected content
	expectedParts := []string{
		"# Test Session",
		"**Provider:** anthropic | **Model:** claude-opus-4",
		"**Date:** February 3, 2026 2:30 PM",
		"**You:**",
		"Hello, how are you?",
		"**Assistant:**",
		"I'm doing well, thank you!",
		"---",
	}

	for _, part := range expectedParts {
		if !strings.Contains(contentStr, part) {
			t.Errorf("Expected content to contain %q, but it didn't.\nFull content:\n%s", part, contentStr)
		}
	}

	// Verify filename format
	expectedFilename := "2026-02-03-test-session.md"
	if !strings.HasSuffix(filePath, expectedFilename) {
		t.Errorf("Expected filename to end with %q, got %q", expectedFilename, filePath)
	}
}

func TestToMarkdown_FileSanitization(t *testing.T) {
	dir := t.TempDir()

	createdAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	session := db.Session{
		ID:        "session-456",
		Title:     "What's the best API? (v2.0)",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: createdAt,
	}

	messages := []db.Message{}

	filePath, err := ToMarkdown(session, messages, dir)
	if err != nil {
		t.Fatalf("ToMarkdown failed: %v", err)
	}

	// Verify filename sanitization
	filename := filepath.Base(filePath)
	expected := "2026-02-03-what-s-the-best-api-v2-0.md"

	if filename != expected {
		t.Errorf("Expected filename %q, got %q", expected, filename)
	}
}

func TestToMarkdown_DirectoryCreation(t *testing.T) {
	baseDir := t.TempDir()
	nestedDir := filepath.Join(baseDir, "exports", "markdown", "2026")

	// Verify directory doesn't exist yet
	if _, err := os.Stat(nestedDir); !os.IsNotExist(err) {
		t.Fatalf("Expected directory to not exist initially")
	}

	session := db.Session{
		ID:        "session-789",
		Title:     "Nested Export",
		Provider:  "anthropic",
		Model:     "claude-sonnet-4",
		CreatedAt: time.Now(),
	}

	messages := []db.Message{}

	filePath, err := ToMarkdown(session, messages, nestedDir)
	if err != nil {
		t.Fatalf("ToMarkdown failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Fatalf("Expected directory to be created at %s", nestedDir)
	}

	// Verify file was created in the nested directory
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist at %s", filePath)
	}
}

func TestToMarkdown_DuplicateFilename(t *testing.T) {
	dir := t.TempDir()

	createdAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	session := db.Session{
		ID:        "session-duplicate",
		Title:     "Duplicate Test",
		Provider:  "anthropic",
		Model:     "claude-opus-4",
		CreatedAt: createdAt,
	}

	messages := []db.Message{}

	// Export first time
	filePath1, err := ToMarkdown(session, messages, dir)
	if err != nil {
		t.Fatalf("First ToMarkdown failed: %v", err)
	}

	// Export second time
	filePath2, err := ToMarkdown(session, messages, dir)
	if err != nil {
		t.Fatalf("Second ToMarkdown failed: %v", err)
	}

	// Verify paths are different
	if filePath1 == filePath2 {
		t.Errorf("Expected different file paths for duplicate exports")
	}

	// Verify first file has no suffix
	expectedFirst := "2026-02-03-duplicate-test.md"
	if !strings.HasSuffix(filePath1, expectedFirst) {
		t.Errorf("Expected first file to be %q, got %q", expectedFirst, filepath.Base(filePath1))
	}

	// Verify second file has -1 suffix
	expectedSecond := "2026-02-03-duplicate-test-1.md"
	if !strings.HasSuffix(filePath2, expectedSecond) {
		t.Errorf("Expected second file to be %q, got %q", expectedSecond, filepath.Base(filePath2))
	}

	// Both files should exist
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Errorf("Expected first file to exist")
	}
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Errorf("Expected second file to exist")
	}
}

func TestToMarkdown_EmptyTitle(t *testing.T) {
	dir := t.TempDir()

	createdAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	session := db.Session{
		ID:        "session-empty",
		Title:     "",
		Provider:  "anthropic",
		Model:     "claude-opus-4",
		CreatedAt: createdAt,
	}

	messages := []db.Message{}

	filePath, err := ToMarkdown(session, messages, dir)
	if err != nil {
		t.Fatalf("ToMarkdown failed: %v", err)
	}

	// Verify filename uses "untitled"
	expected := "2026-02-03-untitled.md"
	if !strings.HasSuffix(filePath, expected) {
		t.Errorf("Expected filename to be %q, got %q", expected, filepath.Base(filePath))
	}
}

func TestToMarkdown_SystemMessagesSkipped(t *testing.T) {
	dir := t.TempDir()

	createdAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	session := db.Session{
		ID:        "session-system",
		Title:     "System Message Test",
		Provider:  "anthropic",
		Model:     "claude-opus-4",
		CreatedAt: createdAt,
	}

	messages := []db.Message{
		{
			ID:        1,
			SessionID: "session-system",
			Role:      "system",
			Content:   "You are a helpful assistant.",
			CreatedAt: createdAt,
			Tokens:    6,
		},
		{
			ID:        2,
			SessionID: "session-system",
			Role:      "user",
			Content:   "Hello!",
			CreatedAt: createdAt.Add(time.Second),
			Tokens:    2,
		},
		{
			ID:        3,
			SessionID: "session-system",
			Role:      "assistant",
			Content:   "Hi there!",
			CreatedAt: createdAt.Add(2 * time.Second),
			Tokens:    3,
		},
	}

	filePath, err := ToMarkdown(session, messages, dir)
	if err != nil {
		t.Fatalf("ToMarkdown failed: %v", err)
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Verify system message content is NOT in the file
	if strings.Contains(contentStr, "You are a helpful assistant") {
		t.Errorf("System message should not appear in output")
	}

	// Verify user and assistant messages ARE in the file
	if !strings.Contains(contentStr, "Hello!") {
		t.Errorf("User message should appear in output")
	}
	if !strings.Contains(contentStr, "Hi there!") {
		t.Errorf("Assistant message should appear in output")
	}
}
