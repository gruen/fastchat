package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClaudeStream_Normal(t *testing.T) {
	// Create test server that returns realistic Claude SSE response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version '2023-06-01', got %q", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Errorf("expected content-type 'application/json', got %q", r.Header.Get("content-type"))
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		response := `event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-3-5-sonnet-20241022","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}
`
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Create provider
	provider := &claudeProvider{
		name:      "test-claude",
		apiKey:    "test-key",
		baseURL:   server.URL,
		model:     "claude-3-5-sonnet-20241022",
		maxTokens: 1024,
		client:    &http.Client{},
	}

	// Stream
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hello"}}
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
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}

	// Check content chunks
	expectedContent := []string{"Hello", " world", "!"}
	for i, expected := range expectedContent {
		if chunks[i].Content != expected {
			t.Errorf("chunk %d: expected content %q, got %q", i, expected, chunks[i].Content)
		}
		if chunks[i].Done {
			t.Errorf("chunk %d: expected Done=false, got true", i)
		}
		if chunks[i].Error != nil {
			t.Errorf("chunk %d: unexpected error: %v", i, chunks[i].Error)
		}
	}

	// Check final chunk
	lastChunk := chunks[len(chunks)-1]
	if !lastChunk.Done {
		t.Error("last chunk: expected Done=true, got false")
	}
	if lastChunk.Content != "" {
		t.Errorf("last chunk: expected empty content, got %q", lastChunk.Content)
	}
	if lastChunk.Error != nil {
		t.Errorf("last chunk: unexpected error: %v", lastChunk.Error)
	}
}

func TestClaudeStream_ErrorResponse(t *testing.T) {
	// Create test server that returns 401 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
	}))
	defer server.Close()

	// Create provider
	provider := &claudeProvider{
		name:      "test-claude",
		apiKey:    "bad-key",
		baseURL:   server.URL,
		model:     "claude-3-5-sonnet-20241022",
		maxTokens: 1024,
		client:    &http.Client{},
	}

	// Stream should return error
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hello"}}
	_, err := provider.Stream(ctx, messages)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify error contains status code
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain '401', got: %v", err)
	}
}

func TestClaudeStream_ContextCancellation(t *testing.T) {
	// Create test server that streams slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send first chunk
		w.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0}\n\n"))
		w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"chunk1\"}}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Wait before sending more
		time.Sleep(100 * time.Millisecond)

		w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"chunk2\"}}\n\n"))
		w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	// Create provider
	provider := &claudeProvider{
		name:      "test-claude",
		apiKey:    "test-key",
		baseURL:   server.URL,
		model:     "claude-3-5-sonnet-20241022",
		maxTokens: 1024,
		client:    &http.Client{},
	}

	// Stream with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	messages := []ChatMessage{{Role: "user", Content: "Hello"}}
	ch, err := provider.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Read first chunk
	chunk := <-ch
	if chunk.Error != nil {
		t.Fatalf("unexpected error: %v", chunk.Error)
	}
	if chunk.Content != "chunk1" {
		t.Errorf("expected content 'chunk1', got %q", chunk.Content)
	}

	// Cancel context
	cancel()

	// Channel should close without hanging
	done := make(chan bool)
	go func() {
		for range ch {
			// Drain remaining chunks
		}
		done <- true
	}()

	select {
	case <-done:
		// Success - channel closed
	case <-time.After(1 * time.Second):
		t.Fatal("channel did not close after context cancellation")
	}
}

func TestClaudeStream_EmptyResponse(t *testing.T) {
	// Create test server that returns message_stop immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		response := `event: message_start
data: {"type":"message_start","message":{"id":"msg_1"}}

event: content_block_start
data: {"type":"content_block_start","index":0}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_stop
data: {"type":"message_stop"}
`
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Create provider
	provider := &claudeProvider{
		name:      "test-claude",
		apiKey:    "test-key",
		baseURL:   server.URL,
		model:     "claude-3-5-sonnet-20241022",
		maxTokens: 1024,
		client:    &http.Client{},
	}

	// Stream
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hello"}}
	ch, err := provider.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Collect chunks
	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should have exactly one chunk (the Done chunk)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	// Verify it's a Done chunk with empty content
	chunk := chunks[0]
	if !chunk.Done {
		t.Error("expected Done=true, got false")
	}
	if chunk.Content != "" {
		t.Errorf("expected empty content, got %q", chunk.Content)
	}
	if chunk.Error != nil {
		t.Errorf("unexpected error: %v", chunk.Error)
	}
}

func TestClaudeStream_APIError(t *testing.T) {
	// Create test server that returns an error event in SSE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		response := `event: error
data: {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}
`
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Create provider
	provider := &claudeProvider{
		name:      "test-claude",
		apiKey:    "test-key",
		baseURL:   server.URL,
		model:     "claude-3-5-sonnet-20241022",
		maxTokens: 1024,
		client:    &http.Client{},
	}

	// Stream
	ctx := context.Background()
	messages := []ChatMessage{{Role: "user", Content: "Hello"}}
	ch, err := provider.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Should get error chunk
	chunk := <-ch
	if chunk.Error == nil {
		t.Fatal("expected error chunk, got nil")
	}
	if !strings.Contains(chunk.Error.Error(), "Overloaded") {
		t.Errorf("expected error to contain 'Overloaded', got: %v", chunk.Error)
	}

	// Channel should close
	_, ok := <-ch
	if ok {
		t.Error("expected channel to close after error")
	}
}
