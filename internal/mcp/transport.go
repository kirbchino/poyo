package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// StdioTransport implements Transport using stdin/stdout
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.Reader
	stderr io.Reader

	mu          sync.Mutex
	started     bool
	onMessage   func([]byte)
	onError     func(error)
	done        chan struct{}
	messageChan chan []byte
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(command string, args []string, env map[string]string) *StdioTransport {
	cmd := exec.Command(command, args...)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	return &StdioTransport{
		cmd:         cmd,
		messageChan: make(chan []byte, 100),
		done:        make(chan struct{}),
	}
}

// Start starts the transport
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("transport already started")
	}

	// Get stdin pipe
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	t.stdin = stdin

	// Get stdout pipe
	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	t.stdout = stdout

	// Get stderr pipe
	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	t.stderr = stderr

	// Start the process
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	t.started = true

	// Start reading stdout
	go t.readLoop()

	// Start reading stderr for debugging
	go t.readStderr()

	return nil
}

// readLoop reads messages from stdout
func (t *StdioTransport) readLoop() {
	reader := bufio.NewReader(t.stdout)
	for {
		select {
		case <-t.done:
			return
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF && t.onError != nil {
					t.onError(fmt.Errorf("read error: %w", err))
				}
				return
			}

			// Remove newline
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}

			// Skip empty lines
			if len(line) == 0 {
				continue
			}

			// Handle message
			if t.onMessage != nil {
				t.onMessage(line)
			} else {
				select {
				case t.messageChan <- line:
				default:
					// Channel full, drop message
				}
			}
		}
	}
}

// readStderr reads from stderr for debugging
func (t *StdioTransport) readStderr() {
	reader := bufio.NewReader(t.stderr)
	for {
		select {
		case <-t.done:
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			// Log stderr for debugging (could be configurable)
			fmt.Fprintf(os.Stderr, "[MCP stderr] %s", line)
		}
	}
}

// Send sends a message
func (t *StdioTransport) Send(message []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return fmt.Errorf("transport not started")
	}

	// Write message with newline
	_, err := t.stdin.Write(append(message, '\n'))
	return err
}

// Receive receives a message
func (t *StdioTransport) Receive() ([]byte, error) {
	select {
	case msg := <-t.messageChan:
		return msg, nil
	case <-t.done:
		return nil, io.EOF
	}
}

// OnMessage sets the message handler
func (t *StdioTransport) OnMessage(handler func([]byte)) {
	t.onMessage = handler
}

// OnError sets the error handler
func (t *StdioTransport) OnError(handler func(error)) {
	t.onError = handler
}

// Close closes the transport
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return nil
	}

	close(t.done)

	// Close stdin
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Kill the process
	if t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}

	t.started = false
	return nil
}

// HTTPTransport implements Transport using HTTP/SSE
type HTTPTransport struct {
	url     string
	headers map[string]string
	client  HTTPClient

	mu        sync.Mutex
	started   bool
	onMessage func([]byte)
	onError   func(error)
	done      chan struct{}
}

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Do(req interface{}) (interface{}, error)
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(url string, headers map[string]string) *HTTPTransport {
	return &HTTPTransport{
		url:     url,
		headers: headers,
		done:    make(chan struct{}),
	}
}

// Start starts the transport
func (t *HTTPTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("transport already started")
	}

	t.started = true
	return nil
}

// Send sends a message via HTTP POST
func (t *HTTPTransport) Send(message []byte) error {
	// TODO: Implement HTTP POST
	return fmt.Errorf("not implemented")
}

// Receive receives a message (polling or SSE)
func (t *HTTPTransport) Receive() ([]byte, error) {
	// TODO: Implement SSE or polling
	return nil, fmt.Errorf("not implemented")
}

// OnMessage sets the message handler
func (t *HTTPTransport) OnMessage(handler func([]byte)) {
	t.onMessage = handler
}

// OnError sets the error handler
func (t *HTTPTransport) OnError(handler func(error)) {
	t.onError = handler
}

// Close closes the transport
func (t *HTTPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	close(t.done)
	t.started = false
	return nil
}

// WebSocketTransport implements Transport using WebSocket
type WebSocketTransport struct {
	url     string
	headers map[string]string
	conn    WebSocketConn

	mu        sync.Mutex
	started   bool
	onMessage func([]byte)
	onError   func(error)
	done      chan struct{}
}

// WebSocketConn interface for WebSocket operations
type WebSocketConn interface {
	Send(message []byte) error
	Receive() ([]byte, error)
	Close() error
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(url string, headers map[string]string) *WebSocketTransport {
	return &WebSocketTransport{
		url:     url,
		headers: headers,
		done:    make(chan struct{}),
	}
}

// Start starts the transport
func (t *WebSocketTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("transport already started")
	}

	// TODO: Implement WebSocket connection
	t.started = true
	return nil
}

// Send sends a message via WebSocket
func (t *WebSocketTransport) Send(message []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return fmt.Errorf("transport not started")
	}

	if t.conn == nil {
		return fmt.Errorf("no connection")
	}

	return t.conn.Send(message)
}

// Receive receives a message via WebSocket
func (t *WebSocketTransport) Receive() ([]byte, error) {
	if t.conn == nil {
		return nil, fmt.Errorf("no connection")
	}
	return t.conn.Receive()
}

// OnMessage sets the message handler
func (t *WebSocketTransport) OnMessage(handler func([]byte)) {
	t.onMessage = handler
}

// OnError sets the error handler
func (t *WebSocketTransport) OnError(handler func(error)) {
	t.onError = handler
}

// Close closes the transport
func (t *WebSocketTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	close(t.done)
	if t.conn != nil {
		t.conn.Close()
	}
	t.started = false
	return nil
}

// InProcessTransport implements Transport for in-process communication
type InProcessTransport struct {
	mu        sync.Mutex
	peer      *InProcessTransport
	onMessage func([]byte)
	onError   func(error)
	closed    bool
}

// NewInProcessTransport creates a new in-process transport
func NewInProcessTransport() *InProcessTransport {
	return &InProcessTransport{}
}

// CreateLinkedTransportPair creates a pair of linked transports
func CreateLinkedTransportPair() (*InProcessTransport, *InProcessTransport) {
	a := NewInProcessTransport()
	b := NewInProcessTransport()
	a.peer = b
	b.peer = a
	return a, b
}

// Start starts the transport
func (t *InProcessTransport) Start(ctx context.Context) error {
	return nil
}

// Send sends a message to the peer
func (t *InProcessTransport) Send(message []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport closed")
	}

	if t.peer != nil && t.peer.onMessage != nil {
		// Deliver asynchronously
		go t.peer.onMessage(message)
	}

	return nil
}

// Receive is not used for in-process transport
func (t *InProcessTransport) Receive() ([]byte, error) {
	return nil, fmt.Errorf("use OnMessage for in-process transport")
}

// OnMessage sets the message handler
func (t *InProcessTransport) OnMessage(handler func([]byte)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onMessage = handler
}

// OnError sets the error handler
func (t *InProcessTransport) OnError(handler func(error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onError = handler
}

// Close closes the transport
func (t *InProcessTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}

// JSONRPCMessage represents a JSON-RPC message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ParseJSONRPCMessage parses a JSON-RPC message
func ParseJSONRPCMessage(data []byte) (*JSONRPCMessage, error) {
	var msg JSONRPCMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC message: %w", err)
	}
	return &msg, nil
}

// EncodeJSONRPCMessage encodes a JSON-RPC message
func EncodeJSONRPCMessage(msg *JSONRPCMessage) ([]byte, error) {
	return json.Marshal(msg)
}
