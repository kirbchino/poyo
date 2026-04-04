// Package tools contains tool definitions and the tool execution framework.
package tools

import (
	"sync"
)

// Registry manages tool registration and lookup.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[tool.Name()] = tool

	// Also register aliases
	for _, alias := range tool.Aliases() {
		r.tools[alias] = tool
	}
}

// Get retrieves a tool by name or alias.
func (r *Registry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tools[name]
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Deduplicate by name
	seen := make(map[string]bool)
	var result []Tool

	for _, tool := range r.tools {
		if !seen[tool.Name()] {
			seen[tool.Name()] = true
			result = append(result, tool)
		}
	}

	return result
}

// Unregister removes a tool from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tool, exists := r.tools[name]
	if !exists {
		return
	}

	// Remove the tool and its aliases
	delete(r.tools, tool.Name())
	for _, alias := range tool.Aliases() {
		delete(r.tools, alias)
	}
}

// Clear removes all tools from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
}

// DefaultRegistry is the global tool registry.
var DefaultRegistry = NewRegistry()

// RegisterTool registers a tool with the default registry.
func RegisterTool(tool Tool) {
	DefaultRegistry.Register(tool)
}

// GetTool retrieves a tool from the default registry.
func GetTool(name string) Tool {
	return DefaultRegistry.Get(name)
}

// GetAllTools returns all tools from the default registry.
func GetAllTools() []Tool {
	return DefaultRegistry.List()
}

// UnregisterTool removes a tool from the default registry.
func UnregisterTool(name string) {
	DefaultRegistry.Unregister(name)
}

// InitializeBuiltinTools registers all built-in tools.
// 🌀 Poyo 的能力系统初始化 - 所有 Copy Ability 在此注册
func InitializeBuiltinTools() {
	// 🔥 Fire - 火焰卡比（命令执行）
	RegisterTool(NewBashTool())

	// 🌀 Inhale - 吸入（文件读取）
	RegisterTool(NewFileReadTool())

	// 💪 Stone - 石头卡比（文件写入）
	RegisterTool(NewFileWriteTool())

	// ⚔️ Sword - 剑士卡比（文件编辑）
	RegisterTool(NewFileEditTool())

	// 🪃 Cutter - 刀片卡比（文件搜索）
	RegisterTool(NewGlobTool())

	// ⚡ Spark - 闪电卡比（内容搜索）
	RegisterTool(NewGrepTool())

	// 🥷 Ninja - 忍者卡比（子代理）
	RegisterTool(NewAgentTool())

	// 📝 Todo - 任务追踪
	RegisterTool(NewTodoWriteTool())

	// 📤 TaskOutput - 任务结果
	taskOutput := NewTaskOutputTool()
	RegisterTool(taskOutput)

	// 🛑 TaskStop - 任务停止
	RegisterTool(NewTaskStopTool(taskOutput))

	// 🌐 WebFetch - 光束获取
	RegisterTool(NewWebFetchTool())

	// 🔎 WebSearch - 光束搜索（真实 API）
	RegisterTool(NewWebSearchTool())

	// 📓 Notebook - 笔记本编辑
	RegisterTool(NewNotebookEditTool())

	// 💬 AskUserQuestion - 用户交互
	RegisterTool(NewAskUserQuestionTool())

	// 📋 EnterPlanMode - 规划模式
	RegisterTool(NewEnterPlanModeTool())

	// ✅ ExitPlanMode - 规划完成
	RegisterTool(NewExitPlanModeTool())

	// ⏰ CronCreate - 定时任务创建
	RegisterTool(NewCronCreateTool())

	// 🗑️ CronDelete - 定时任务删除
	RegisterTool(NewCronDeleteTool())

	// 📅 CronList - 定时任务列表
	RegisterTool(NewCronListTool())

	// 🌿 EnterWorktree - Git Worktree 进入
	RegisterTool(NewEnterWorktreeTool())

	// 🚪 ExitWorktree - Git Worktree 退出
	RegisterTool(NewExitWorktreeTool())

	// ⭐ Skill - Skill 工具
	RegisterTool(NewSkillTool())

	// 📸 MediaRead - 图片和 PDF 读取
	RegisterTool(NewMediaReadTool())
}
