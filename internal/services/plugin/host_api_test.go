package plugin

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════
// Lua Plugin Host API Tests
// ═══════════════════════════════════════════════════════

// MockToolExecutor implements ToolExecutor for testing
type MockToolExecutor struct {
	results map[string]interface{}
	errors  map[string]error
}

func NewMockToolExecutor() *MockToolExecutor {
	return &MockToolExecutor{
		results: make(map[string]interface{}),
		errors:  make(map[string]error),
	}
}

func (m *MockToolExecutor) ExecuteTool(ctx context.Context, name string, input interface{}) (interface{}, error) {
	if err, ok := m.errors[name]; ok {
		return nil, err
	}
	if result, ok := m.results[name]; ok {
		return result, nil
	}
	return map[string]interface{}{"tool": name, "executed": true}, nil
}

func (m *MockToolExecutor) SetResult(name string, result interface{}) {
	m.results[name] = result
}

func (m *MockToolExecutor) SetError(name string, err error) {
	m.errors[name] = err
}

// MockCacheHandler implements CacheHandler for testing
type MockCacheHandler struct {
	data  map[string]interface{}
	mu    sync.RWMutex
}

func NewMockCacheHandler() *MockCacheHandler {
	return &MockCacheHandler{
		data: make(map[string]interface{}),
	}
}

func (m *MockCacheHandler) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

func (m *MockCacheHandler) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *MockCacheHandler) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *MockCacheHandler) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]interface{})
}

// MockContextHandler implements ContextHandler for testing
type MockContextHandler struct {
	conversationID string
	userID         string
	context        map[string]interface{}
}

func NewMockContextHandler() *MockContextHandler {
	return &MockContextHandler{
		context: make(map[string]interface{}),
	}
}

func (m *MockContextHandler) GetContext() map[string]interface{} {
	return m.context
}

func (m *MockContextHandler) SetContext(key string, value interface{}) {
	m.context[key] = value
}

func (m *MockContextHandler) GetConversationID() string {
	return m.conversationID
}

func (m *MockContextHandler) GetUserID() string {
	return m.userID
}

// ═══════════════════════════════════════════════════════
// Test: Lua Plugin Loading
// ═══════════════════════════════════════════════════════

func TestLuaPluginLoad(t *testing.T) {
	// Create test script
	testScript := `
local M = {}
function M.init()
    poyo.log("info", "Plugin loaded")
end
return M
`

	// Write test script
	scriptPath := "/tmp/test/main.lua"
	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	handler := NewLuaPluginHandler("/tmp/test", NewDefaultLuaAPI("/tmp/test"), NewMockToolExecutor())
	handler.SetCacheHandler(NewMockCacheHandler())
	handler.SetContextHandler(NewMockContextHandler())

	plugin := &Plugin{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		Type:    PluginTypeLua,
		Main:    "main.lua",
		Path:    "/tmp/test",
		Config: map[string]interface{}{
			"debug": true,
		},
	}

	ctx := context.Background()
	err := handler.Load(ctx, plugin)
	if err != nil {
		t.Fatalf("Failed to load plugin: %v", err)
	}

	err = handler.Unload(ctx, plugin)
	if err != nil {
		t.Fatalf("Failed to unload plugin: %v", err)
	}
}

// ═══════════════════════════════════════════════════════
// Test: Tool Execution from Plugin
// ═══════════════════════════════════════════════════════

func TestLuaPluginToolExecution(t *testing.T) {
	mockExec := NewMockToolExecutor()
	mockExec.SetResult("Read", map[string]interface{}{
		"content":  "test content",
		"lines":    1,
		"language": "text",
	})

	handler := NewLuaPluginHandler("/tmp/test", NewDefaultLuaAPI("/tmp/test"), mockExec)

	plugin := &Plugin{
		ID:      "test-tool",
		Name:    "Tool Test",
		Version: "1.0.0",
		Type:    PluginTypeLua,
		Main:    "test_tool.lua",
		Path:    "/tmp/test",
	}

	// Create test script
	testScript := `
function test_read()
    local result, err = poyo.use("Read", {path = "test.txt"})
    if err then
        return {success = false, error = err}
    end
    return {success = true, content = result.content}
end
return {test_read = test_read}
`

	// Write test script
	scriptPath := "/tmp/test/test_tool.lua"
	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	ctx := context.Background()
	if err := handler.Load(ctx, plugin); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer handler.Unload(ctx, plugin)

	// Execute the test function
	result, err := handler.Execute(ctx, plugin, "test_read", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if !resultMap["success"].(bool) {
		t.Errorf("Expected success, got: %v", resultMap)
	}
}

// ═══════════════════════════════════════════════════════
// Test: File System Operations from Plugin
// ═══════════════════════════════════════════════════════

func TestLuaPluginFileOperations(t *testing.T) {
	handler := NewLuaPluginHandler("/tmp/test", NewDefaultLuaAPI("/tmp/test"), NewMockToolExecutor())

	plugin := &Plugin{
		ID:      "test-fs",
		Name:    "FS Test",
		Version: "1.0.0",
		Type:    PluginTypeLua,
		Main:    "test_fs.lua",
		Path:    "/tmp/test",
	}

	testScript := `
function test_fs()
    -- Write file
    local ok, err = poyo.fs.write("test_output.txt", "Hello from plugin!")
    if not ok then
        return {success = false, stage = "write", error = err}
    end

    -- Check exists
    local exists = poyo.fs.exists("test_output.txt")
    if not exists then
        return {success = false, stage = "exists"}
    end

    -- Read file
    local content, err = poyo.fs.read("test_output.txt")
    if err then
        return {success = false, stage = "read", error = err}
    end

    -- Cleanup
    poyo.fs.remove("test_output.txt")

    return {
        success = true,
        content_match = content == "Hello from plugin!"
    }
end
return {test_fs = test_fs}
`

	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/test/test_fs.lua", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	ctx := context.Background()
	if err := handler.Load(ctx, plugin); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer handler.Unload(ctx, plugin)

	result, err := handler.Execute(ctx, plugin, "test_fs", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if !resultMap["success"].(bool) {
		t.Errorf("Test failed at stage: %v", resultMap["stage"])
	}
	if !resultMap["content_match"].(bool) {
		t.Error("Content mismatch")
	}
}

// ═══════════════════════════════════════════════════════
// Test: JSON Operations from Plugin
// ═══════════════════════════════════════════════════════

func TestLuaPluginJSONOperations(t *testing.T) {
	handler := NewLuaPluginHandler("/tmp/test", NewDefaultLuaAPI("/tmp/test"), NewMockToolExecutor())

	plugin := &Plugin{
		ID:      "test-json",
		Name:    "JSON Test",
		Version: "1.0.0",
		Type:    PluginTypeLua,
		Main:    "test_json.lua",
		Path:    "/tmp/test",
	}

	testScript := `
function test_json()
    local data = {
        name = "poyo",
        version = "1.0",
        features = {"tools", "hooks"}
    }

    -- Encode
    local encoded, err = poyo.json.encode(data)
    if err then
        return {success = false, stage = "encode", error = err}
    end

    -- Decode
    local decoded, err = poyo.json.decode(encoded)
    if err then
        return {success = false, stage = "decode", error = err}
    end

    -- Pretty
    local pretty, err = poyo.json.pretty(data)
    if err then
        return {success = false, stage = "pretty", error = err}
    end

    return {
        success = true,
        name_match = decoded.name == "poyo",
        version_match = decoded.version == "1.0",
        has_newlines = string.find(pretty, "\n") ~= nil
    }
end
return {test_json = test_json}
`

	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/test/test_json.lua", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	ctx := context.Background()
	if err := handler.Load(ctx, plugin); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer handler.Unload(ctx, plugin)

	result, err := handler.Execute(ctx, plugin, "test_json", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if !resultMap["success"].(bool) {
		t.Errorf("Test failed: %v", resultMap)
	}
	if !resultMap["name_match"].(bool) {
		t.Error("Name mismatch after decode")
	}
}

// ═══════════════════════════════════════════════════════
// Test: Cache Operations from Plugin
// ═══════════════════════════════════════════════════════

func TestLuaPluginCacheOperations(t *testing.T) {
	cacheHandler := NewMockCacheHandler()
	handler := NewLuaPluginHandler("/tmp/test", NewDefaultLuaAPI("/tmp/test"), NewMockToolExecutor())
	handler.SetCacheHandler(cacheHandler)

	plugin := &Plugin{
		ID:      "test-cache",
		Name:    "Cache Test",
		Version: "1.0.0",
		Type:    PluginTypeLua,
		Main:    "test_cache.lua",
		Path:    "/tmp/test",
	}

	testScript := `
function test_cache()
    local key = "test_key_" .. os.time()
    local value = {name = "poyo", count = 42}

    -- Set
    poyo.cache.set(key, value, 60)

    -- Get
    local cached, found = poyo.cache.get(key)
    if not found then
        return {success = false, stage = "get"}
    end

    -- Delete
    poyo.cache.delete(key)

    -- Verify deleted
    local _, still_found = poyo.cache.get(key)
    if still_found then
        return {success = false, stage = "delete"}
    end

    return {
        success = true,
        name_match = cached.name == "poyo",
        count_match = cached.count == 42
    }
end
return {test_cache = test_cache}
`

	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/test/test_cache.lua", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	ctx := context.Background()
	if err := handler.Load(ctx, plugin); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer handler.Unload(ctx, plugin)

	result, err := handler.Execute(ctx, plugin, "test_cache", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if !resultMap["success"].(bool) {
		t.Errorf("Test failed at stage: %v", resultMap["stage"])
	}
}

// ═══════════════════════════════════════════════════════
// Test: Environment Variables from Plugin
// ═══════════════════════════════════════════════════════

func TestLuaPluginEnvOperations(t *testing.T) {
	api := NewDefaultLuaAPI("/tmp/test")
	handler := NewLuaPluginHandler("/tmp/test", api, NewMockToolExecutor())

	plugin := &Plugin{
		ID:      "test-env",
		Name:    "Env Test",
		Version: "1.0.0",
		Type:    PluginTypeLua,
		Main:    "test_env.lua",
		Path:    "/tmp/test",
	}

	testScript := `
function test_env()
    -- Set env
    poyo.env.set("POYO_TEST_VAR", "test_value")

    -- Get env
    local value = poyo.env.get("POYO_TEST_VAR")
    if value ~= "test_value" then
        return {success = false, stage = "get", value = value}
    end

    -- List env
    local all_env = poyo.env.list()
    local has_test = all_env["POYO_TEST_VAR"] == "test_value"

    return {
        success = true,
        get_match = value == "test_value",
        list_has_var = has_test
    }
end
return {test_env = test_env}
`

	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/test/test_env.lua", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	ctx := context.Background()
	if err := handler.Load(ctx, plugin); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer handler.Unload(ctx, plugin)

	result, err := handler.Execute(ctx, plugin, "test_env", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if !resultMap["success"].(bool) {
		t.Errorf("Test failed: %v", resultMap)
	}
}

// ═══════════════════════════════════════════════════════
// Test: Hook Execution with Host API Access
// ═══════════════════════════════════════════════════════

func TestLuaPluginHookWithHostAPI(t *testing.T) {
	cacheHandler := NewMockCacheHandler()
	mockExec := NewMockToolExecutor()
	handler := NewLuaPluginHandler("/tmp/test", NewDefaultLuaAPI("/tmp/test"), mockExec)
	handler.SetCacheHandler(cacheHandler)

	plugin := &Plugin{
		ID:      "test-hook",
		Name:    "Hook Test",
		Version: "1.0.0",
		Type:    PluginTypeLua,
		Main:    "test_hook.lua",
		Path:    "/tmp/test",
	}

	testScript := `
function PreToolUse(input)
    -- Access host API from hook
    local tool = input.tool or input.args and input.args.tool or "unknown"
    poyo.log("info", "Hook monitoring: " .. tool)

    -- Use cache from hook
    local key = "hook_count_" .. tool
    local count, _ = poyo.cache.get(key)
    count = (count or 0) + 1
    poyo.cache.set(key, count)

    return {
        blocked = false,
        message = "Hook executed",
        tool_count = count
    }
end
return {PreToolUse = PreToolUse}
`

	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/test/test_hook.lua", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	ctx := context.Background()
	if err := handler.Load(ctx, plugin); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer handler.Unload(ctx, plugin)

	// Execute hook
	result, err := handler.Execute(ctx, plugin, "PreToolUse", map[string]interface{}{
		"tool": "Bash",
		"args": map[string]interface{}{
			"command": "echo test",
		},
	})
	if err != nil {
		t.Fatalf("Hook execute failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["blocked"].(bool) {
		t.Error("Hook should not block")
	}

	// Verify cache was updated
	count, found := cacheHandler.Get("hook_count_Bash")
	if !found {
		t.Error("Cache should have hook count")
	}
	if count.(float64) != 1 {
		t.Errorf("Expected count 1, got %v", count)
	}
}

// ═══════════════════════════════════════════════════════
// Test: Plugin Info Access
// ═══════════════════════════════════════════════════════

func TestLuaPluginInfoAccess(t *testing.T) {
	handler := NewLuaPluginHandler("/tmp/test", NewDefaultLuaAPI("/tmp/test"), NewMockToolExecutor())

	plugin := &Plugin{
		ID:      "test-info",
		Name:    "Info Test Plugin",
		Version: "2.0.0",
		Type:    PluginTypeLua,
		Main:    "test_info.lua",
		Path:    "/tmp/test",
		Config: map[string]interface{}{
			"debug":  true,
			"mode":   "test",
			"retries": 3,
		},
	}

	testScript := `
function get_info()
    return {
        plugin_id = poyo.plugin.id,
        plugin_name = poyo.plugin.name,
        plugin_version = poyo.plugin.version,
        plugin_path = poyo.plugin.path,
        dream_land = poyo.land,
        config_debug = poyo.config.debug,
        config_mode = poyo.config.mode,
        config_retries = poyo.config.retries
    }
end
return {get_info = get_info}
`

	if err := os.MkdirAll("/tmp/test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/test/test_info.lua", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/tmp/test")

	ctx := context.Background()
	if err := handler.Load(ctx, plugin); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer handler.Unload(ctx, plugin)

	result, err := handler.Execute(ctx, plugin, "get_info", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	info := result.(map[string]interface{})

	if info["plugin_id"] != "test-info" {
		t.Errorf("Expected plugin_id 'test-info', got %v", info["plugin_id"])
	}
	if info["plugin_name"] != "Info Test Plugin" {
		t.Errorf("Expected plugin_name 'Info Test Plugin', got %v", info["plugin_name"])
	}
	if info["plugin_version"] != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got %v", info["plugin_version"])
	}
	if info["config_debug"].(bool) != true {
		t.Error("Expected config.debug to be true")
	}
	if info["config_mode"] != "test" {
		t.Errorf("Expected config.mode 'test', got %v", info["config_mode"])
	}
}
