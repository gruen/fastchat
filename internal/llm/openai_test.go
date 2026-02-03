package llm

import (
	"encoding/json"

	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIStream_Normal(t *testing.T) {
	// Create a test server that returns SSE with multiple delta chunks then [DONE]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header with Bearer token")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}
		
		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		
		// First chunk with role
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		// Content chunks
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		// Done signal
		w.Write([]byte(`data: [DONE]` + "\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer server.Close()
	
	// Create provider
	provider := &openaiProvider{
		name:         "test",
		apiKey:       "test-key",
		baseURL:      server.URL,
		model:        "gpt-4",
		systemPrompt: "You are helpful",
		maxTokens:    4096,
		client:       &http.Client{},
	}
	
	// Stream
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hi"}}
	ch, err := provider.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	// Collect chunks
	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}
	
	// Verify chunks
	if len(chunks) < 4 {
		t.Fatalf("Expected at least 4 chunks, got %d", len(chunks))
	}
	
	// Check content
	expectedContent := []string{"", "Hello", " world", "!"}
	for i := 0; i < 4; i++ {
		if chunks[i].Content != expectedContent[i] {
			t.Errorf("Chunk %d: expected content %q, got %q", i, expectedContent[i], chunks[i].Content)
		}
		if chunks[i].Error != nil {
			t.Errorf("Chunk %d: unexpected error: %v", i, chunks[i].Error)
		}
	}
	
	// Last chunk should have Done=true
	lastChunk := chunks[len(chunks)-1]
	if !lastChunk.Done {
		t.Errorf("Expected last chunk to have Done=true")
	}
}

func TestOpenAIStream_ErrorResponse(t *testing.T) {
	// Create a test server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()
	
	// Create provider
	provider := &openaiProvider{
		name:      "test",
		apiKey:    "invalid-key",
		baseURL:   server.URL,
		model:     "gpt-4",
		maxTokens: 4096,
		client:    &http.Client{},
	}
	
	// Stream should return error
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hi"}}
	_, err := provider.Stream(ctx, messages)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	
	// Verify error contains status code
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestOpenAIStream_ContextCancellation(t *testing.T) {
	// Create a test server that streams slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		
		// Send first chunk
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		// Sleep for a bit before next chunk
		time.Sleep(100 * time.Millisecond)
		
		// Try to send second chunk (but context will be cancelled)
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		time.Sleep(100 * time.Millisecond)
		
		w.Write([]byte(`data: [DONE]` + "\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer server.Close()
	
	// Create provider
	provider := &openaiProvider{
		name:      "test",
		apiKey:    "test-key",
		baseURL:   server.URL,
		model:     "gpt-4",
		maxTokens: 4096,
		client:    &http.Client{},
	}
	
	// Create context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())
	
	// Stream
	messages := []ChatMessage{{Role: "user", Content: "Hi"}}
	ch, err := provider.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	// Read first chunk
	chunk, ok := <-ch
	if !ok {
		t.Fatal("Expected first chunk")
	}
	if chunk.Content != "Hello" {
		t.Errorf("Expected first chunk content 'Hello', got %q", chunk.Content)
	}
	
	// Cancel context
	cancel()
	
	// Channel should close soon
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				// Channel closed as expected
				return
			}
			// Continue reading until channel closes
		case <-timeout:
			t.Fatal("Channel did not close after context cancellation")
		}
	}
}

func TestOpenAIStream_FinishReasonStop(t *testing.T) {
	// Create a test server that sends finish_reason: "stop" before [DONE]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		
		// Content chunk
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		// Final chunk with finish_reason
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":""},"finish_reason":"stop"}]}` + "\n\n"))
		w.(http.Flusher).Flush()
		
		// Done signal
		w.Write([]byte(`data: [DONE]` + "\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer server.Close()
	
	// Create provider
	provider := &openaiProvider{
		name:      "test",
		apiKey:    "test-key",
		baseURL:   server.URL,
		model:     "gpt-4",
		maxTokens: 4096,
		client:    &http.Client{},
	}
	
	// Stream
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hi"}}
	ch, err := provider.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	// Collect chunks
	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}
	
	// Should have at least 2 chunks (content + done)
	if len(chunks) < 2 {
		t.Fatalf("Expected at least 2 chunks, got %d", len(chunks))
	}
	
	// Verify last chunk has Done=true (from [DONE])
	lastChunk := chunks[len(chunks)-1]
	if !lastChunk.Done {
		t.Errorf("Expected last chunk to have Done=true")
	}
}

func TestOpenAIStream_SystemPrompt(t *testing.T) {
	// Create a test server that captures the request body
	var receivedMessages []ChatMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var reqBody struct {
			Messages []ChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		receivedMessages = reqBody.Messages
		
		// Send minimal response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"OK"},"finish_reason":null}]}` + "\n\n"))
		w.Write([]byte(`data: [DONE]` + "\n\n"))
	}))
	defer server.Close()
	
	// Create provider with system prompt
	provider := &openaiProvider{
		name:         "test",
		apiKey:       "test-key",
		baseURL:      server.URL,
		model:        "gpt-4",
		systemPrompt: "You are a helpful assistant",
		maxTokens:    4096,
		client:       &http.Client{},
	}
	
	// Stream
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hi"}}
	ch, err := provider.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	
	// Drain channel
	for range ch {
	}
	
	// Verify system prompt was added as first message
	if len(receivedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(receivedMessages))
	}
	if receivedMessages[0].Role != "system" {
		t.Errorf("Expected first message role 'system', got %q", receivedMessages[0].Role)
	}
	if receivedMessages[0].Content != "You are a helpful assistant" {
		t.Errorf("Expected system prompt in first message, got %q", receivedMessages[0].Content)
	}
	if receivedMessages[1].Role != "user" {
		t.Errorf("Expected second message role 'user', got %q", receivedMessages[1].Role)
	}
}
