package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type claudeProvider struct {
	name         string
	apiKey       string
	baseURL      string
	model        string
	systemPrompt string
	maxTokens    int
	client       *http.Client
}

func (p *claudeProvider) Name() string {
	return p.name
}

func (p *claudeProvider) Stream(ctx context.Context, messages []ChatMessage) (<-chan StreamChunk, error) {
	// Build request body
	reqBody := map[string]interface{}{
		"model":      p.model,
		"max_tokens": p.maxTokens,
		"stream":     true,
		"messages":   messages,
	}
	if p.systemPrompt != "" {
		reqBody["system"] = p.systemPrompt
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := p.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Create output channel
	ch := make(chan StreamChunk, 1)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// Use ParseSSE to handle the SSE stream
		sseChannel := ParseSSE(ctx, resp.Body, func(data []byte) (StreamChunk, bool) {
			// Parse the JSON data
			var event map[string]interface{}
			if err := json.Unmarshal(data, &event); err != nil {
				return StreamChunk{Error: fmt.Errorf("failed to parse SSE data: %w", err)}, true
			}

			eventType, ok := event["type"].(string)
			if !ok {
				return StreamChunk{}, false // Ignore malformed events
			}

			switch eventType {
			case "content_block_delta":
				// Extract delta.text
				delta, ok := event["delta"].(map[string]interface{})
				if !ok {
					return StreamChunk{}, false
				}
				text, ok := delta["text"].(string)
				if !ok {
					return StreamChunk{}, false
				}
				return StreamChunk{Content: text, Done: false}, false

			case "message_stop":
				return StreamChunk{Done: true}, true

			case "error":
				// Extract error information
				errMsg := "unknown error"
				if errData, ok := event["error"].(map[string]interface{}); ok {
					if msg, ok := errData["message"].(string); ok {
						errMsg = msg
					}
				}
				return StreamChunk{Error: fmt.Errorf("API error: %s", errMsg)}, true

			default:
				// Ignore other event types (message_start, content_block_start, etc.)
				return StreamChunk{}, false
			}
		})

		// Filter and forward chunks
		for chunk := range sseChannel {
			// Only send chunks that have content, are done, or have an error
			if chunk.Content != "" || chunk.Done || chunk.Error != nil {
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}
