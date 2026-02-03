package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mg/ai-tui/internal/config"
)

// StreamChunk represents one piece of a streaming LLM response.
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// Provider is the interface all LLM backends implement.
type Provider interface {
	// Stream sends messages and returns a channel of StreamChunks.
	// The channel is closed when the response is complete or context is cancelled.
	Stream(ctx context.Context, messages []ChatMessage) (<-chan StreamChunk, error)

	// Name returns the provider name from config.
	Name() string
}

// BuildProviders creates Provider instances from config.
// If base_url contains "anthropic.com", creates a Claude provider.
// Otherwise creates an OpenAI-compatible provider.
func BuildProviders(providers map[string]config.Provider) map[string]Provider {
	result := make(map[string]Provider)
	for name, cfg := range providers {
		if strings.Contains(cfg.BaseURL, "anthropic.com") {
			result[name] = &claudeProvider{
				name:         name,
				apiKey:       cfg.APIKey,
				baseURL:      cfg.BaseURL,
				model:        cfg.Model,
				systemPrompt: cfg.SystemPrompt,
				maxTokens:    cfg.MaxTokens,
				client:       &http.Client{},
			}
		} else {
			result[name] = &openaiProvider{
				name:         name,
				apiKey:       cfg.APIKey,
				baseURL:      cfg.BaseURL,
				model:        cfg.Model,
				systemPrompt: cfg.SystemPrompt,
				maxTokens:    cfg.MaxTokens,
				client:       &http.Client{},
			}
		}
	}
	return result
}

// claudeProvider stub - will be implemented in claude.go by the other agent
type claudeProvider struct {
	name         string
	apiKey       string
	baseURL      string
	model        string
	systemPrompt string
	maxTokens    int
	client       *http.Client
}

func (p *claudeProvider) Stream(ctx context.Context, messages []ChatMessage) (<-chan StreamChunk, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *claudeProvider) Name() string {
	return p.name
}
