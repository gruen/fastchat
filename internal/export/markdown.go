package export

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mg/ai-tui/internal/db"
)

// ToMarkdown exports a session and its messages to a markdown file in dir.
// Returns the full path of the created file.
func ToMarkdown(session db.Session, messages []db.Message, dir string) (string, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate filename
	datePrefix := session.CreatedAt.Format("2006-01-02")
	sanitizedTitle := sanitizeTitle(session.Title)
	baseFilename := fmt.Sprintf("%s-%s.md", datePrefix, sanitizedTitle)

	// Handle duplicate filenames
	filename := baseFilename
	counter := 1
	for {
		fullPath := filepath.Join(dir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		filename = fmt.Sprintf("%s-%s-%d.md", datePrefix, sanitizedTitle, counter)
		counter++
	}

	fullPath := filepath.Join(dir, filename)

	// Build markdown content
	content := buildMarkdownContent(session, messages)

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fullPath, nil // fallback to relative path if abs fails
	}

	return absPath, nil
}

// sanitizeTitle converts a title into a safe filename component
func sanitizeTitle(title string) string {
	if title == "" {
		return "untitled"
	}

	// Convert to lowercase
	s := strings.ToLower(title)

	// Replace spaces and non-alphanumeric chars with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	// Collapse multiple hyphens
	reg = regexp.MustCompile(`-+`)
	s = reg.ReplaceAllString(s, "-")

	// Trim hyphens from edges
	s = strings.Trim(s, "-")

	// Max 50 chars
	if len(s) > 50 {
		s = s[:50]
		// Trim any trailing hyphen after truncation
		s = strings.TrimRight(s, "-")
	}

	// Fallback if sanitization resulted in empty string
	if s == "" {
		return "untitled"
	}

	return s
}

// buildMarkdownContent generates the markdown content from session and messages
func buildMarkdownContent(session db.Session, messages []db.Message) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# %s\n\n", session.Title))

	// Metadata
	sb.WriteString(fmt.Sprintf("**Provider:** %s | **Model:** %s  \n", session.Provider, session.Model))
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", session.CreatedAt.Format("January 2, 2006 3:04 PM")))

	// Messages
	for _, msg := range messages {
		// Skip system messages
		if msg.Role == "system" {
			continue
		}

		sb.WriteString("---\n\n")

		// Role header
		if msg.Role == "user" {
			sb.WriteString("**You:**\n\n")
		} else if msg.Role == "assistant" {
			sb.WriteString("**Assistant:**\n\n")
		}

		// Content
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	// Final separator
	sb.WriteString("---\n")

	return sb.String()
}
