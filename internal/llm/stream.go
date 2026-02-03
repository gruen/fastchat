package llm

import (
	"bufio"
	"context"
	"io"
	"strings"
)

// ParseSSE reads Server-Sent Events from body and sends parsed data to the returned channel.
// The onData callback receives the raw data bytes (after "data: " prefix) and returns
// a StreamChunk and a bool indicating if the stream should stop.
// The channel is closed when: body is exhausted, context is cancelled, or onData signals stop.
func ParseSSE(ctx context.Context, body io.ReadCloser, onData func(data []byte) (StreamChunk, bool)) <-chan StreamChunk {
	ch := make(chan StreamChunk, 1)

	go func() {
		defer close(ch)
		defer body.Close()

		scanner := bufio.NewScanner(body)
		for scanner.Scan() {
			// Check if context was cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				continue
			}

			// Skip SSE comments
			if strings.HasPrefix(line, ":") {
				continue
			}

			// Process data lines
			if strings.HasPrefix(line, "data: ") {
				data := []byte(strings.TrimPrefix(line, "data: "))
				chunk, stop := onData(data)

				// Try to send the chunk, but respect context cancellation
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}

				// If onData signals stop, close the stream
				if stop {
					return
				}
			}
		}

		// Check for scanner errors (but don't send them if context was cancelled)
		if err := scanner.Err(); err != nil {
			select {
			case ch <- StreamChunk{Error: err}:
			case <-ctx.Done():
			}
		}
	}()

	return ch
}
