package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type openaiProvider struct {
	name         string
	apiKey       string
	baseURL      string
	model        string
	systemPrompt string
	maxTokens    int
	client       *http.Client
}

func (p *openaiProvider) Name() string {
	return p.name
}

func (p *openaiProvider) Stream(ctx context.Context, messages []ChatMessage) (<-chan StreamChunk, error) {
	// Build the request body
	reqMessages := make([]ChatMessage, 0, len(messages)+1)
	
	// Add system prompt as the first message if present
	if p.systemPrompt != "" {
		reqMessages = append(reqMessages, ChatMessage{
			Role:    "system",
			Content: p.systemPrompt,
		})
	}
	
	// Add the conversation messages
	reqMessages = append(reqMessages, messages...)
	
	reqBody := map[string]interface{}{
		"model":      p.model,
		"max_tokens": p.maxTokens,
		"stream":     true,
		"messages":   reqMessages,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create the HTTP request
	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	// Send the request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	// Parse SSE stream
	return ParseSSE(ctx, resp.Body, p.parseChunk), nil
}

// parseChunk processes a single SSE data line and returns the chunk and whether to stop
func (p *openaiProvider) parseChunk(data []byte) (StreamChunk, bool) {
	// Check for [DONE] signal
	if string(data) == "[DONE]" {
		return StreamChunk{Done: true}, true
	}
	
	// Parse the JSON response
	var response struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	}
	
	if err := json.Unmarshal(data, &response); err != nil {
		return StreamChunk{Error: fmt.Errorf("failed to parse chunk: %w", err)}, true
	}
	
	// Extract content from the first choice
	if len(response.Choices) > 0 {
		content := response.Choices[0].Delta.Content
		finishReason := response.Choices[0].FinishReason
		
		// If we have a finish_reason, this is the last content chunk
		if finishReason != nil && *finishReason != "" {
			return StreamChunk{Content: content, Done: false}, false
		}
		
		return StreamChunk{Content: content}, false
	}
	
	// Empty chunk
	return StreamChunk{}, false
}
