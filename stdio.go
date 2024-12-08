// This file implements the stdio transport layer for JSON-RPC communication.
// It provides functionality to read and write JSON-RPC messages over standard input/output
// streams, similar to the TypeScript implementation in @typescript-sdk/src/shared/stdio.ts.
//
// Key Components:
//
// 1. ReadBuffer:
//   - Buffers continuous stdio stream into discrete JSON-RPC messages
//   - Thread-safe with mutex protection
//   - Handles message framing using newline delimiters
//   - Methods: Append (add data), ReadMessage (read complete message), Clear (reset buffer)
//
// 2. StdioTransport:
//   - Implements the Transport interface using stdio
//   - Uses bufio.Reader for efficient buffered reading
//   - Thread-safe with mutex protection
//   - Supports:
//   - Asynchronous message reading
//   - Message sending with newline framing
//   - Proper cleanup on close
//   - Event handlers for close, error, and message events
//
// 3. Message Handling:
//   - Deserializes JSON-RPC messages into appropriate types:
//   - JSONRPCRequest: Messages with ID and method
//   - JSONRPCNotification: Messages with method but no ID
//   - JSONRPCError: Error responses with ID
//   - Generic responses: Success responses with ID
//   - Serializes messages to JSON with newline termination
//
// Thread Safety:
//   - All public methods are thread-safe
//   - Uses sync.Mutex for state protection
//   - Safe for concurrent reading and writing
//
// Usage:
//
//	transport := NewStdioTransport()
//	transport.SetMessageHandler(func(msg interface{}) {
//	    // Handle incoming message
//	})
//	transport.Start()
//	defer transport.Close()
//
//	// Send a message
//	transport.Send(map[string]interface{}{
//	    "jsonrpc": "2.0",
//	    "method": "test",
//	    "params": map[string]interface{}{},
//	})
//
// Error Handling:
//   - All methods return meaningful errors
//   - Transport supports error handler for async errors
//   - Proper cleanup on error conditions
//
// For more details, see the test file stdio_test.go.
package mcp

import (
	"encoding/json"
	"errors"
	"github.com/davecgh/go-spew/spew"
	"sync"
)

// ReadBuffer buffers a continuous stdio stream into discrete JSON-RPC messages.
type ReadBuffer struct {
	mu     sync.Mutex
	buffer []byte
}

// NewReadBuffer creates a new ReadBuffer.
func NewReadBuffer() *ReadBuffer {
	return &ReadBuffer{}
}

// Append adds a chunk of data to the buffer.
func (rb *ReadBuffer) Append(chunk []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.buffer == nil {
		rb.buffer = chunk
	} else {
		rb.buffer = append(rb.buffer, chunk...)
	}
}

// ReadMessage reads a complete JSON-RPC message from the buffer.
// Returns nil if no complete message is available.
func (rb *ReadBuffer) ReadMessage() (*BaseMessage, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.buffer == nil {
		return nil, nil
	}

	// Find newline
	for i := 0; i < len(rb.buffer); i++ {
		if rb.buffer[i] == '\n' {
			// Extract line
			line := string(rb.buffer[:i])
			rb.buffer = rb.buffer[i+1:]
			println("serialized message:", line)
			return deserializeMessage(line)
		}
	}

	return nil, nil
}

// Clear clears the buffer.
func (rb *ReadBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.buffer = nil
}

// deserializeMessage deserializes a JSON-RPC message from a string.
func deserializeMessage(line string) (*BaseMessage, error) {
	var request BaseJSONRPCRequest
	if err := json.Unmarshal([]byte(line), &request); err == nil {
		println("unmarshaled request:", spew.Sdump(request))
		return NewBaseMessageRequest(request), nil
	} else {
		println("unmarshaled request error:", err.Error())
	}

	var notification BaseJSONRPCNotification
	if err := json.Unmarshal([]byte(line), &notification); err == nil {
		return NewBaseMessageNotification(notification), nil
	}

	// TODO: Add error handling and response deserialization

	// Must be a response
	return nil, errors.New("failed to unmarshal JSON-RPC message, unrecognized type")
}
