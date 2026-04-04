// Package plugin provides Lua plugin support for Poyo
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// ToolExecutor defines the interface for executing tools from plugins
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, toolName string, input interface{}) (interface{}, error)
}

// PromptHandler defines the interface for prompting users
type PromptHandler interface {
	Prompt(message string, options []string) (string, error)
	PromptInput(message string, defaultValue string) (string, error)
	Confirm(message string, defaultValue bool) (bool, error)
}

// CacheHandler defines the interface for caching
type CacheHandler interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

// ContextHandler defines the interface for context access
type ContextHandler interface {
	GetContext() map[string]interface{}
	SetContext(key string, value interface{})
	GetConversationID() string
	GetUserID() string
}

// LuaPluginHandler handles Lua-based plugins with embedded Lua VM
type LuaPluginHandler struct {
	mu            sync.RWMutex
	vms           map[string]*lua.LState
	workingDir    string
	api           LuaAPI
	toolExec      ToolExecutor
	promptHandler PromptHandler
	cacheHandler  CacheHandler
	ctxHandler    ContextHandler
	ctx           context.Context
}

// LuaAPI defines the API exposed to Lua scripts
type LuaAPI interface {
	GetWorkingDir() string
	GetEnv(key string) string
	SetEnv(key, value string)
	Log(level, message string)
}

// NewLuaPluginHandler creates a new Lua plugin handler
func NewLuaPluginHandler(workingDir string, api LuaAPI, toolExec ToolExecutor) *LuaPluginHandler {
	return &LuaPluginHandler{
		vms:        make(map[string]*lua.LState),
		workingDir: workingDir,
		api:        api,
		toolExec:   toolExec,
		ctx:        context.Background(),
	}
}

// SetPromptHandler sets the prompt handler
func (h *LuaPluginHandler) SetPromptHandler(handler PromptHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.promptHandler = handler
}

// SetCacheHandler sets the cache handler
func (h *LuaPluginHandler) SetCacheHandler(handler CacheHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cacheHandler = handler
}

// SetContextHandler sets the context handler
func (h *LuaPluginHandler) SetContextHandler(handler ContextHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ctxHandler = handler
}

// SetContext sets the context for tool execution
func (h *LuaPluginHandler) SetContext(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ctx = ctx
}

// Load loads a Lua plugin
func (h *LuaPluginHandler) Load(ctx context.Context, plugin *Plugin) error {
	if plugin.Main == "" {
		return fmt.Errorf("no main script specified")
	}

	scriptPath := filepath.Join(plugin.Path, plugin.Main)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	L := lua.NewState()
	L.OpenLibs()
	h.registerAPI(L, plugin)

	if err := L.DoFile(scriptPath); err != nil {
		L.Close()
		return fmt.Errorf("failed to load Lua script: %w", err)
	}

	h.mu.Lock()
	h.vms[plugin.ID] = L
	h.mu.Unlock()

	return nil
}

// Unload unloads a Lua plugin
func (h *LuaPluginHandler) Unload(ctx context.Context, plugin *Plugin) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if vm, ok := h.vms[plugin.ID]; ok {
		vm.Close()
		delete(h.vms, plugin.ID)
	}
	return nil
}

// Execute executes a Lua plugin method
func (h *LuaPluginHandler) Execute(ctx context.Context, plugin *Plugin, method string, input interface{}) (interface{}, error) {
	h.mu.RLock()
	L, ok := h.vms[plugin.ID]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %s not loaded", plugin.ID)
	}

	fn := L.GetGlobal(method)
	if fn.Type() != lua.LTFunction {
		return nil, fmt.Errorf("method %s not found or not a function", method)
	}

	inputValue, err := h.goToLua(L, input)
	if err != nil {
		return nil, fmt.Errorf("convert input: %w", err)
	}

	L.Push(fn)
	L.Push(inputValue)

	if err := L.PCall(1, 1, nil); err != nil {
		return nil, fmt.Errorf("Lua execution error: %w", err)
	}

	result := L.Get(-1)
	L.Pop(1)

	return h.luaToGo(result), nil
}

// registerAPI 注册完整的 poyo API
// 💚 全部功能都在 poyo 命名空间下，简单直观
func (h *LuaPluginHandler) registerAPI(L *lua.LState, plugin *Plugin) {
	poyo := L.NewTable()
	L.SetGlobal("poyo", poyo)

	// ─────────────────────────────────────────────────────────
	// 插件信息
	// ─────────────────────────────────────────────────────────
	pluginTable := L.NewTable()
	L.SetField(pluginTable, "id", lua.LString(plugin.ID))
	L.SetField(pluginTable, "name", lua.LString(plugin.Name))
	L.SetField(pluginTable, "version", lua.LString(plugin.Version))
	L.SetField(pluginTable, "path", lua.LString(plugin.Path))
	L.SetField(poyo, "plugin", pluginTable)

	// 配置
	configTable := L.NewTable()
	for k, v := range plugin.Config {
		val, _ := h.goToLua(L, v)
		L.SetField(configTable, k, val)
	}
	L.SetField(poyo, "config", configTable)

	// ─────────────────────────────────────────────────────────
	// 梦之国（工作目录、环境、会话）
	// ─────────────────────────────────────────────────────────
	if h.api != nil {
		L.SetField(poyo, "land", lua.LString(h.api.GetWorkingDir()))
	} else {
		L.SetField(poyo, "land", lua.LString(h.workingDir))
	}

	// poyo.env - 环境变量
	envTable := L.NewTable()
	L.SetField(poyo, "env", envTable)
	L.SetField(envTable, "get", L.NewFunction(func(L *lua.LState) int {
		key := L.ToString(1)
		var val string
		if h.api != nil {
			val = h.api.GetEnv(key)
		}
		L.Push(lua.LString(val))
		return 1
	}))
	L.SetField(envTable, "set", L.NewFunction(func(L *lua.LState) int {
		if h.api != nil {
			h.api.SetEnv(L.ToString(1), L.ToString(2))
		}
		return 0
	}))
	L.SetField(envTable, "list", L.NewFunction(func(L *lua.LState) int {
		result := L.NewTable()
		for _, e := range os.Environ() {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				L.SetField(result, parts[0], lua.LString(parts[1]))
			}
		}
		L.Push(result)
		return 1
	}))

	// poyo.session - 会话信息
	sessionTable := L.NewTable()
	L.SetField(poyo, "session", sessionTable)
	L.SetField(sessionTable, "id", L.NewFunction(func(L *lua.LState) int {
		if h.ctxHandler != nil {
			L.Push(lua.LString(h.ctxHandler.GetConversationID()))
			return 1
		}
		L.Push(lua.LString(""))
		return 1
	}))
	L.SetField(sessionTable, "user_id", L.NewFunction(func(L *lua.LState) int {
		if h.ctxHandler != nil {
			L.Push(lua.LString(h.ctxHandler.GetUserID()))
			return 1
		}
		L.Push(lua.LString(""))
		return 1
	}))

	// poyo.context - 上下文
	ctxTable := L.NewTable()
	L.SetField(poyo, "context", ctxTable)
	L.SetField(ctxTable, "get", L.NewFunction(func(L *lua.LState) int {
		if h.ctxHandler != nil {
			c, _ := h.goToLua(L, h.ctxHandler.GetContext())
			L.Push(c)
			return 1
		}
		L.Push(L.NewTable())
		return 1
	}))
	L.SetField(ctxTable, "set", L.NewFunction(func(L *lua.LState) int {
		if h.ctxHandler != nil {
			h.ctxHandler.SetContext(L.ToString(1), h.luaToGo(L.Get(2)))
		}
		return 0
	}))

	// ─────────────────────────────────────────────────────────
	// 日志
	// ─────────────────────────────────────────────────────────
	L.SetField(poyo, "log", L.NewFunction(func(L *lua.LState) int {
		level, msg := L.ToString(1), L.ToString(2)
		if h.api != nil {
			h.api.Log(level, msg)
		} else {
			fmt.Printf("[%s] %s\n", level, msg)
		}
		return 0
	}))
	L.SetField(poyo, "debug", L.NewFunction(func(L *lua.LState) int {
		if h.api != nil {
			h.api.Log("DEBUG", L.ToString(1))
		}
		return 0
	}))
	L.SetField(poyo, "info", L.NewFunction(func(L *lua.LState) int {
		if h.api != nil {
			h.api.Log("INFO", L.ToString(1))
		}
		return 0
	}))
	L.SetField(poyo, "warn", L.NewFunction(func(L *lua.LState) int {
		if h.api != nil {
			h.api.Log("WARN", L.ToString(1))
		}
		return 0
	}))
	L.SetField(poyo, "error", L.NewFunction(func(L *lua.LState) int {
		if h.api != nil {
			h.api.Log("ERROR", L.ToString(1))
		}
		return 0
	}))

	// ─────────────────────────────────────────────────────────
	// JSON
	// ─────────────────────────────────────────────────────────
	jsonTable := L.NewTable()
	L.SetField(poyo, "json", jsonTable)
	L.SetField(jsonTable, "encode", L.NewFunction(func(L *lua.LState) int {
		b, err := json.Marshal(h.luaToGo(L.Get(1)))
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(string(b)))
		return 1
	}))
	L.SetField(jsonTable, "decode", L.NewFunction(func(L *lua.LState) int {
		var v interface{}
		if err := json.Unmarshal([]byte(L.ToString(1)), &v); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		lv, _ := h.goToLua(L, v)
		L.Push(lv)
		return 1
	}))
	L.SetField(jsonTable, "pretty", L.NewFunction(func(L *lua.LState) int {
		b, err := json.MarshalIndent(h.luaToGo(L.Get(1)), "", "  ")
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(string(b)))
		return 1
	}))

	// ─────────────────────────────────────────────────────────
	// HTTP
	// ─────────────────────────────────────────────────────────
	httpTable := L.NewTable()
	L.SetField(poyo, "http", httpTable)
	L.SetField(httpTable, "get", L.NewFunction(func(L *lua.LState) int {
		resp, err := h.doHTTP("GET", L.ToString(1), h.parseHeaders(L.Get(2)), nil)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		lv, _ := h.goToLua(L, resp)
		L.Push(lv)
		return 1
	}))
	L.SetField(httpTable, "post", L.NewFunction(func(L *lua.LState) int {
		body, _ := json.Marshal(h.luaToGo(L.Get(2)))
		headers := h.parseHeaders(L.Get(3))
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
		resp, err := h.doHTTP("POST", L.ToString(1), headers, body)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		lv, _ := h.goToLua(L, resp)
		L.Push(lv)
		return 1
	}))

	// ─────────────────────────────────────────────────────────
	// 文件系统
	// ─────────────────────────────────────────────────────────
	fsTable := L.NewTable()
	L.SetField(poyo, "fs", fsTable)
	L.SetField(fsTable, "read", L.NewFunction(func(L *lua.LState) int {
		path := h.resolvePath(L.ToString(1), plugin.Path)
		b, err := os.ReadFile(path)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(string(b)))
		return 1
	}))
	L.SetField(fsTable, "write", L.NewFunction(func(L *lua.LState) int {
		path := h.resolvePath(L.ToString(1), plugin.Path)
		if err := os.WriteFile(path, []byte(L.ToString(2)), 0644); err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	}))
	L.SetField(fsTable, "exists", L.NewFunction(func(L *lua.LState) int {
		_, err := os.Stat(h.resolvePath(L.ToString(1), plugin.Path))
		L.Push(lua.LBool(err == nil))
		return 1
	}))
	L.SetField(fsTable, "list", L.NewFunction(func(L *lua.LState) int {
		entries, err := os.ReadDir(h.resolvePath(L.ToString(1), plugin.Path))
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		t := L.NewTable()
		for i, e := range entries {
			et := L.NewTable()
			L.SetField(et, "name", lua.LString(e.Name()))
			L.SetField(et, "is_dir", lua.LBool(e.IsDir()))
			L.SetTable(t, lua.LNumber(i+1), et)
		}
		L.Push(t)
		return 1
	}))
	L.SetField(fsTable, "mkdir", L.NewFunction(func(L *lua.LState) int {
		if err := os.MkdirAll(h.resolvePath(L.ToString(1), plugin.Path), 0755); err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	}))
	L.SetField(fsTable, "remove", L.NewFunction(func(L *lua.LState) int {
		path := h.resolvePath(L.ToString(1), plugin.Path)
		var err error
		if info, e := os.Stat(path); e == nil && info.IsDir() {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
		if err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	}))

	// ─────────────────────────────────────────────────────────
	// 用户交互
	// ─────────────────────────────────────────────────────────
	promptTable := L.NewTable()
	L.SetField(poyo, "prompt", promptTable)
	L.SetField(promptTable, "select", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		var opts []string
		if L.Get(2).Type() == lua.LTTable {
			L.Get(2).(*lua.LTable).ForEach(func(_, v lua.LValue) {
				if v.Type() == lua.LTString {
					opts = append(opts, string(v.(lua.LString)))
				}
			})
		}
		if h.promptHandler != nil {
			choice, err := h.promptHandler.Prompt(msg, opts)
			if err != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			L.Push(lua.LString(choice))
			return 1
		}
		if len(opts) > 0 {
			L.Push(lua.LString(opts[0]))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))
	L.SetField(promptTable, "input", L.NewFunction(func(L *lua.LState) int {
		if h.promptHandler != nil {
			input, err := h.promptHandler.PromptInput(L.ToString(1), L.ToString(2))
			if err != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			L.Push(lua.LString(input))
			return 1
		}
		L.Push(lua.LString(L.ToString(2)))
		return 1
	}))
	L.SetField(promptTable, "confirm", L.NewFunction(func(L *lua.LState) int {
		if h.promptHandler != nil {
			ok, err := h.promptHandler.Confirm(L.ToString(1), L.ToBool(2))
			if err != nil {
				L.Push(lua.LFalse)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			L.Push(lua.LBool(ok))
			return 1
		}
		L.Push(lua.LBool(L.ToBool(2)))
		return 1
	}))

	// ─────────────────────────────────────────────────────────
	// 缓存
	// ─────────────────────────────────────────────────────────
	cacheTable := L.NewTable()
	L.SetField(poyo, "cache", cacheTable)
	L.SetField(cacheTable, "get", L.NewFunction(func(L *lua.LState) int {
		if h.cacheHandler != nil {
			v, ok := h.cacheHandler.Get(L.ToString(1))
			if !ok {
				L.Push(lua.LNil)
				L.Push(lua.LFalse)
				return 2
			}
			lv, _ := h.goToLua(L, v)
			L.Push(lv)
			L.Push(lua.LBool(true))
			return 2
		}
		L.Push(lua.LNil)
		L.Push(lua.LFalse)
		return 2
	}))
	L.SetField(cacheTable, "set", L.NewFunction(func(L *lua.LState) int {
		ttl := time.Duration(0)
		if L.GetTop() >= 3 {
			ttl = time.Duration(L.ToInt(3)) * time.Second
		}
		if h.cacheHandler != nil {
			h.cacheHandler.Set(L.ToString(1), h.luaToGo(L.Get(2)), ttl)
		}
		return 0
	}))
	L.SetField(cacheTable, "delete", L.NewFunction(func(L *lua.LState) int {
		if h.cacheHandler != nil {
			h.cacheHandler.Delete(L.ToString(1))
		}
		return 0
	}))
	L.SetField(cacheTable, "clear", L.NewFunction(func(L *lua.LState) int {
		if h.cacheHandler != nil {
			h.cacheHandler.Clear()
		}
		return 0
	}))

	// ─────────────────────────────────────────────────────────
	// 钩子
	// ─────────────────────────────────────────────────────────
	hookTable := L.NewTable()
	L.SetField(poyo, "hook", hookTable)
	L.SetField(hookTable, "register", L.NewFunction(func(L *lua.LState) int { return 0 }))
	L.SetField(hookTable, "types", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		for _, typ := range []string{"PreToolUse", "PostToolUse", "PrePrompt", "PostPrompt", "OnError"} {
			L.SetTable(t, lua.LString(typ), lua.LBool(true))
		}
		L.Push(t)
		return 1
	}))

	// ─────────────────────────────────────────────────────────
	// 命令
	// ─────────────────────────────────────────────────────────
	cmdTable := L.NewTable()
	L.SetField(poyo, "command", cmdTable)
	L.SetField(cmdTable, "register", L.NewFunction(func(L *lua.LState) int { return 0 }))
	L.SetField(cmdTable, "list", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		for _, cmd := range []string{"/help", "/clear", "/model", "/ability"} {
			L.SetTable(t, lua.LString(cmd), lua.LBool(true))
		}
		L.Push(t)
		return 1
	}))

	// ─────────────────────────────────────────────────────────
	// 能力系统（工具调用）
	// ─────────────────────────────────────────────────────────
	abilityTable := L.NewTable()
	L.SetField(poyo, "ability", abilityTable)

	// poyo.ability.use(name, input)
	L.SetField(abilityTable, "use", L.NewFunction(func(L *lua.LState) int {
		return h.executeAbility(L, L.ToString(1), L.Get(2))
	}))
	L.SetField(abilityTable, "list", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		for _, a := range []string{"Read", "Write", "Edit", "Bash", "Grep", "Glob"} {
			L.SetTable(t, lua.LString(a), lua.LBool(true))
		}
		L.Push(t)
		return 1
	}))
	// poyo.ability.copy(enemy) - 复制能力
	L.SetField(abilityTable, "copy", L.NewFunction(func(L *lua.LState) int {
		enemy := L.ToString(1)
		fmt.Printf("⭐ Poyo copies %s ability!\n", enemy)
		L.Push(lua.LBool(true))
		L.Push(lua.LString("Copied " + enemy))
		return 2
	}))

	// 快捷方式：poyo.use(name, input)
	L.SetField(poyo, "use", L.NewFunction(func(L *lua.LState) int {
		return h.executeAbility(L, L.ToString(1), L.Get(2))
	}))

	// poyo.copy(enemy) - 快捷方式
	L.SetField(poyo, "copy", L.NewFunction(func(L *lua.LState) int {
		enemy := L.ToString(1)
		fmt.Printf("⭐ Poyo copies %s ability!\n", enemy)
		L.Push(lua.LBool(true))
		return 1
	}))

	// ─────────────────────────────────────────────────────────
	// 有趣的 API 💚
	// ─────────────────────────────────────────────────────────
	L.SetField(poyo, "say", L.NewFunction(func(L *lua.LState) int {
		fmt.Printf("💚 Poyo says: %s\n", L.ToString(1))
		return 0
	}))
	L.SetField(poyo, "dance", L.NewFunction(func(L *lua.LState) int {
		fmt.Println("💚 Poyo dances! ♪(๑ᴖ◡ᴖ๑)♪")
		return 0
	}))
	L.SetField(poyo, "poyo", L.NewFunction(func(L *lua.LState) int {
		fmt.Println("💚 Poyo~! ☆彡")
		return 0
	}))
	L.SetField(poyo, "inhale", L.NewFunction(func(L *lua.LState) int {
		fmt.Printf("🌀 Poyo inhales: %s\n", L.ToString(1))
		L.Push(lua.LBool(true))
		return 1
	}))
}

// executeAbility 执行能力（工具）
func (h *LuaPluginHandler) executeAbility(L *lua.LState, name string, input lua.LValue) int {
	if h.toolExec == nil {
		L.Push(lua.LNil)
		L.Push(lua.LString("ability executor not configured"))
		return 2
	}

	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := h.toolExec.ExecuteTool(ctx, name, h.luaToGo(input))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	lv, _ := h.goToLua(L, result)
	L.Push(lv)
	return 1
}

// resolvePath 解析路径
func (h *LuaPluginHandler) resolvePath(path, pluginPath string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(pluginPath, path)
}

// parseHeaders 解析 HTTP 头
func (h *LuaPluginHandler) parseHeaders(param lua.LValue) map[string]string {
	headers := make(map[string]string)
	if param.Type() == lua.LTTable {
		param.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			headers[string(k.(lua.LString))] = string(v.(lua.LString))
		})
	}
	return headers
}

// HTTPResponse HTTP 响应
type HTTPResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

func (h *LuaPluginHandler) doHTTP(method, url string, headers map[string]string, body []byte) (*HTTPResponse, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, strings.NewReader(string(body)))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       string(respBody),
	}, nil
}

// goToLua Go 值转 Lua 值
func (h *LuaPluginHandler) goToLua(L *lua.LState, value interface{}) (lua.LValue, error) {
	if value == nil {
		return lua.LNil, nil
	}

	switch v := value.(type) {
	case bool:
		return lua.LBool(v), nil
	case int:
		return lua.LNumber(v), nil
	case int64:
		return lua.LNumber(v), nil
	case float64:
		return lua.LNumber(v), nil
	case string:
		return lua.LString(v), nil
	case []interface{}:
		t := L.NewTable()
		for i, item := range v {
			lv, err := h.goToLua(L, item)
			if err != nil {
				return nil, err
			}
			L.SetTable(t, lua.LNumber(i+1), lv)
		}
		return t, nil
	case map[string]interface{}:
		t := L.NewTable()
		for k, item := range v {
			lv, err := h.goToLua(L, item)
			if err != nil {
				return nil, err
			}
			L.SetField(t, k, lv)
		}
		return t, nil
	case *HTTPResponse:
		return h.goToLua(L, map[string]interface{}{
			"statusCode": v.StatusCode,
			"headers":    v.Headers,
			"body":       v.Body,
		})
	default:
		b, err := json.Marshal(value)
		if err != nil {
			return lua.LNil, fmt.Errorf("unsupported type: %T", value)
		}
		var generic interface{}
		json.Unmarshal(b, &generic)
		return h.goToLua(L, generic)
	}
}

// luaToGo Lua 值转 Go 值
func (h *LuaPluginHandler) luaToGo(value lua.LValue) interface{} {
	switch value.Type() {
	case lua.LTNil:
		return nil
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTNumber:
		return float64(value.(lua.LNumber))
	case lua.LTString:
		return string(value.(lua.LString))
	case lua.LTTable:
		t := value.(*lua.LTable)
		isArray := true
		maxIdx := 0
		t.ForEach(func(k, _ lua.LValue) {
			if k.Type() != lua.LTNumber {
				isArray = false
			} else {
				if idx := int(k.(lua.LNumber)); idx > maxIdx {
					maxIdx = idx
				}
			}
		})

		if isArray && maxIdx > 0 {
			arr := make([]interface{}, maxIdx)
			t.ForEach(func(k, v lua.LValue) {
				if idx := int(k.(lua.LNumber)) - 1; idx >= 0 && idx < maxIdx {
					arr[idx] = h.luaToGo(v)
				}
			})
			return arr
		}

		m := make(map[string]interface{})
		t.ForEach(func(k, v lua.LValue) {
			var key string
			switch k.Type() {
			case lua.LTString:
				key = string(k.(lua.LString))
			case lua.LTNumber:
				key = fmt.Sprintf("%d", int(k.(lua.LNumber)))
			default:
				key = fmt.Sprintf("%v", k)
			}
			m[key] = h.luaToGo(v)
		})
		return m
	default:
		return nil
	}
}

// Close 关闭所有 VM
func (h *LuaPluginHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, vm := range h.vms {
		vm.Close()
	}
	h.vms = make(map[string]*lua.LState)
	return nil
}

// DefaultLuaAPI 默认 LuaAPI 实现
type DefaultLuaAPI struct {
	workingDir string
	env        map[string]string
}

// NewDefaultLuaAPI 创建默认 LuaAPI
func NewDefaultLuaAPI(workingDir string) *DefaultLuaAPI {
	return &DefaultLuaAPI{
		workingDir: workingDir,
		env:        make(map[string]string),
	}
}

func (a *DefaultLuaAPI) GetWorkingDir() string    { return a.workingDir }
func (a *DefaultLuaAPI) GetEnv(key string) string { return a.env[key] }
func (a *DefaultLuaAPI) SetEnv(key, value string) { a.env[key] = value }
func (a *DefaultLuaAPI) Log(level, message string) {
	fmt.Printf("[%s] %s\n", level, message)
}
