// Package prompt provides centralized prompt management for Poyo.
//
// This package contains all prompts, messages, and descriptions used throughout
// the Poyo application, designed with a Kirby-themed personality.
//
// # POYO Identity
//
// P-O-Y-O represents:
//   - P = Portal     — 所有 Agent 的统一入口
//   - O = Orchestrator — 协调插件、能力、工具的执行
//   - Y = Yield      — 生成结果、交付价值
//   - O = Omnibus    — 兼容所有类型的 Agent 插件
//
// # Structure
//
// The package is organized into several files:
//   - prompts.go  — Core identity and personality
//   - system.go   — System prompts for different modes
//   - tools.go    — Tool descriptions (Copy Abilities)
//   - messages.go — Message templates
//   - hooks.go    — Hook and plugin descriptions
//   - tui.go      — Terminal UI prompts
//
// # Usage
//
// Import the package and use the provided functions:
//
//	import "github.com/kirbchino/poyo/internal/prompt"
//
//	// Get system prompt for a mode
//	sysPrompt := prompt.GetSystemPrompt("interactive")
//
//	// Get tool description
//	desc := prompt.GetToolDescription("Bash")
//
//	// Generate a quick message
//	msg := prompt.MsgProgressDone()
//
//	// Get a fun greeting
//	greeting := prompt.PoyoGreeting()
//
// # Kirby Theme
//
// All prompts are designed to feel like Kirby's Copy Abilities:
//   - 🌀 Inhale  — Read/understand operations
//   - ⭐ Copy     — Learn and replicate patterns
//   - 💪 Ability — Tool usage
//   - 🌙 Dream Land — Working environment
package prompt
