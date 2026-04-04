package mcp

import (
	"context"
	"testing"
	"time"
)

func TestTransportType(t *testing.T) {
	types := []TransportType{
		TransportStdio,
		TransportSSE,
		TransportHTTP,
		TransportWS,
		TransportSDK,
	}

	for _, tt := range types {
		if string(tt) == "" {
			t.Errorf("Transport type should have a non-empty string representation")
		}
	}
}

func TestConnectionState(t *testing.T) {
	states := []ConnectionState{
		StatePending,
		StateConnected,
		StateFailed,
		StateDisabled,
		StateNeedsAuth,
	}

	for _, s := range states {
		if string(s) == "" {
			t.Errorf("Connection state should have a non-empty string representation")
		}
	}
}

func TestConfigScope(t *testing.T) {
	scopes := []ConfigScope{
		ScopeLocal,
		ScopeUser,
		ScopeProject,
		ScopeDynamic,
		ScopeEnterprise,
		ScopeManaged,
	}

	for _, s := range scopes {
		if string(s) == "" {
			t.Errorf("Config scope should have a non-empty string representation")
		}
	}
}

func TestNewConnectionManager(t *testing.T) {
	cm := NewConnectionManager()
	if cm == nil {
		t.Fatal("NewConnectionManager() returned nil")
	}

	if cm.connections == nil {
		t.Error("connections map should be initialized")
	}

	if cm.tools == nil {
		t.Error("tools map should be initialized")
	}
}

func TestNewManager(t *testing.T) {
	options := DefaultMCPOptions()
	m := NewManager(options)

	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.connections == nil {
		t.Error("connections map should be initialized")
	}

	if m.options.MaxReconnectAttempts != 5 {
		t.Errorf("Expected MaxReconnectAttempts 5, got %d", m.options.MaxReconnectAttempts)
	}
}

func TestManagerAddConfig(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	config := &ServerConfig{
		Type:    TransportStdio,
		Command: "test-server",
		Args:    []string{"--port", "8080"},
	}

	m.AddConfig("test-server", config)

	if _, ok := m.configs["test-server"]; !ok {
		t.Error("Config should be added")
	}
}

func TestManagerRemoveConfig(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	config := &ServerConfig{
		Type:    TransportStdio,
		Command: "test-server",
	}

	m.AddConfig("test-server", config)
	m.RemoveConfig("test-server")

	if _, ok := m.configs["test-server"]; ok {
		t.Error("Config should be removed")
	}
}

func TestManagerGetTools(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	// Add a tool directly
	m.mu.Lock()
	m.tools["mcp__test__tool1"] = &Tool{
		Name:       "mcp__test__tool1",
		IsMCP:      true,
		ServerName: "test",
	}
	m.mu.Unlock()

	tools := m.GetTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
}

func TestManagerGetTool(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	m.mu.Lock()
	m.tools["test-tool"] = &Tool{Name: "test-tool"}
	m.mu.Unlock()

	tool, ok := m.GetTool("test-tool")
	if !ok {
		t.Error("Tool should be found")
	}
	if tool.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got %q", tool.Name)
	}

	_, ok = m.GetTool("nonexistent")
	if ok {
		t.Error("Nonexistent tool should not be found")
	}
}

func TestManagerGetConnections(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	m.mu.Lock()
	m.connections["test"] = &ServerConnection{
		Name:  "test",
		State: StateConnected,
	}
	m.mu.Unlock()

	conns := m.GetConnections()
	if len(conns) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(conns))
	}
}

func TestGetMCPPrefix(t *testing.T) {
	tests := []struct {
		serverName string
		expected   string
	}{
		{"test-server", "mcp__test-server__"},
		{"My Server", "mcp__My_Server__"},
		{"server-123", "mcp__server-123__"},
	}

	for _, tt := range tests {
		result := GetMCPPrefix(tt.serverName)
		if result != tt.expected {
			t.Errorf("GetMCPPrefix(%q) = %q, want %q", tt.serverName, result, tt.expected)
		}
	}
}

func TestNormalizeNameForMCP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test-server", "test-server"},
		{"My Server", "My_Server"},
		{"server@123!", "server_123"},
		{"multiple---dashes", "multiple-dashes"},
	}

	for _, tt := range tests {
		result := NormalizeNameForMCP(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeNameForMCP(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsMCPTool(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"mcp__server__tool", true},
		{"Bash", false},
		{"Read", false},
		{"mcp_test_tool", false},
	}

	for _, tt := range tests {
		result := IsMCPTool(tt.name)
		if result != tt.expected {
			t.Errorf("IsMCPTool(%q) = %v, want %v", tt.name, result, tt.expected)
		}
	}
}

func TestParseMCPToolName(t *testing.T) {
	tests := []struct {
		name         string
		expectServer string
		expectTool   string
		expectOK     bool
	}{
		{"mcp__server__tool", "server", "tool", true},
		{"mcp__my-server__some_tool", "my-server", "some_tool", true},
		{"Bash", "", "", false},
		{"mcp_invalid", "", "", false},
	}

	for _, tt := range tests {
		server, tool, ok := ParseMCPToolName(tt.name)
		if ok != tt.expectOK {
			t.Errorf("ParseMCPToolName(%q) ok = %v, want %v", tt.name, ok, tt.expectOK)
		}
		if ok {
			if server != tt.expectServer {
				t.Errorf("ParseMCPToolName(%q) server = %q, want %q", tt.name, server, tt.expectServer)
			}
			if tool != tt.expectTool {
				t.Errorf("ParseMCPToolName(%q) tool = %q, want %q", tt.name, tool, tt.expectTool)
			}
		}
	}
}

func TestDefaultMCPOptions(t *testing.T) {
	opts := DefaultMCPOptions()

	if opts.MaxReconnectAttempts != 5 {
		t.Errorf("Expected MaxReconnectAttempts 5, got %d", opts.MaxReconnectAttempts)
	}

	if opts.InitialBackoff != 1*time.Second {
		t.Errorf("Expected InitialBackoff 1s, got %v", opts.InitialBackoff)
	}

	if opts.MaxBackoff != 30*time.Second {
		t.Errorf("Expected MaxBackoff 30s, got %v", opts.MaxBackoff)
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient(DefaultMCPOptions())
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.pending == nil {
		t.Error("pending map should be initialized")
	}
}

func TestNewStdioTransport(t *testing.T) {
	transport := NewStdioTransport("echo", []string{"hello"}, nil)
	if transport == nil {
		t.Fatal("NewStdioTransport() returned nil")
	}

	if transport.cmd == nil {
		t.Error("cmd should be set")
	}
}

func TestNewHTTPTransport(t *testing.T) {
	transport := NewHTTPTransport("http://localhost:8080", nil)
	if transport == nil {
		t.Fatal("NewHTTPTransport() returned nil")
	}

	if transport.url != "http://localhost:8080" {
		t.Errorf("Expected url 'http://localhost:8080', got %q", transport.url)
	}
}

func TestNewWebSocketTransport(t *testing.T) {
	transport := NewWebSocketTransport("ws://localhost:8080", nil)
	if transport == nil {
		t.Fatal("NewWebSocketTransport() returned nil")
	}

	if transport.url != "ws://localhost:8080" {
		t.Errorf("Expected url 'ws://localhost:8080', got %q", transport.url)
	}
}

func TestCreateLinkedTransportPair(t *testing.T) {
	client, server := CreateLinkedTransportPair()
	if client == nil || server == nil {
		t.Fatal("CreateLinkedTransportPair() returned nil")
	}

	if client.peer != server {
		t.Error("Client peer should be server")
	}

	if server.peer != client {
		t.Error("Server peer should be client")
	}
}

func TestInProcessTransportSend(t *testing.T) {
	client, server := CreateLinkedTransportPair()

	received := make(chan []byte, 1)
	server.OnMessage(func(data []byte) {
		received <- data
	})

	ctx := context.Background()
	_ = client.Start(ctx)
	_ = server.Start(ctx)

	testData := []byte(`{"test":"message"}`)
	if err := client.Send(testData); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	select {
	case msg := <-received:
		if string(msg) != string(testData) {
			t.Errorf("Received %q, want %q", string(msg), string(testData))
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestJSONRPCMessage(t *testing.T) {
	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	}

	data, err := EncodeJSONRPCMessage(msg)
	if err != nil {
		t.Fatalf("EncodeJSONRPCMessage() error: %v", err)
	}

	parsed, err := ParseJSONRPCMessage(data)
	if err != nil {
		t.Fatalf("ParseJSONRPCMessage() error: %v", err)
	}

	if parsed.JSONRPC != msg.JSONRPC {
		t.Errorf("JSONRPC = %q, want %q", parsed.JSONRPC, msg.JSONRPC)
	}

	if parsed.Method != msg.Method {
		t.Errorf("Method = %q, want %q", parsed.Method, msg.Method)
	}
}

func TestManagerClose(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	// Add a mock connection
	m.mu.Lock()
	m.connections["test"] = &ServerConnection{
		Name:  "test",
		State: StateConnected,
	}
	m.mu.Unlock()

	if err := m.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}

	if len(m.connections) != 0 {
		t.Error("Connections should be cleared after Close()")
	}
}

func TestManagerDisconnect(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	// Add a mock connection with tools
	m.mu.Lock()
	m.connections["test"] = &ServerConnection{
		Name:  "test",
		State: StateConnected,
	}
	m.tools["mcp__test__tool1"] = &Tool{
		Name:       "mcp__test__tool1",
		ServerName: "test",
	}
	m.mu.Unlock()

	if err := m.Disconnect("test"); err != nil {
		t.Errorf("Disconnect() error: %v", err)
	}

	if _, ok := m.connections["test"]; ok {
		t.Error("Connection should be removed")
	}

	if _, ok := m.tools["mcp__test__tool1"]; ok {
		t.Error("Tools should be removed")
	}
}

func TestManagerOnEvent(t *testing.T) {
	m := NewManager(DefaultMCPOptions())

	eventReceived := make(chan ConnectionEvent, 1)
	m.OnEvent(func(event ConnectionEvent) {
		eventReceived <- event
	})

	// Trigger an event
	m.emitEvent(ConnectionEvent{
		ServerName: "test",
		State:      StateConnected,
	})

	select {
	case event := <-eventReceived:
		if event.ServerName != "test" {
			t.Errorf("Event ServerName = %q, want 'test'", event.ServerName)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event")
	}
}
