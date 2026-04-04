package prompt

// TUI (Terminal User Interface) related prompts and messages
var TUI = struct {
	// Welcome and help
	Welcome  string
	HelpText string

	// Status messages
	StatusReady    string
	StatusWorking  string
	StatusThinking string
	StatusError    string

	// Input prompts
	InputPrompt string
	InputHelp   string

	// Tool panel
	PanelTools   string
	PanelFiles   string
	PanelHistory string

	// Keyboard shortcuts
	Shortcuts string
}{
	Welcome: `
💚 ═════════════════════════════════════════════════
   🌀 Poyo - Portal Of Your Orchestrated Omnibus-agents
   🌙 欢迎来到梦之国！
══════════════════════════════════════════════════ 💚

就像星之卡比能吸入并复制敌人的能力一样，
Poyo 可以帮助你完成各种代码任务！

💡 输入 /help 查看帮助，或直接告诉 Poyo 你想做什么`,

	HelpText: `
💚 Poyo 帮助中心

📍 快捷命令:
  /help      - 显示帮助
  /clear     - 清空对话
  /exit      - 退出 Poyo
  /session   - 会话管理
  /plugin    - 插件管理
  /ability   - 查看能力列表

🔧 能力系统:
  /ability list    - 列出所有可用能力
  /ability use     - 使用指定能力

🔌 插件系统:
  /plugin list     - 列出已安装插件
  /plugin install  - 安装新插件

📝 会话管理:
  /session save    - 保存当前会话
  /session load    - 加载历史会话
  /session list    - 列出所有会话
`,

	StatusReady:    "💚 Poyo 准备就绪",
	StatusWorking:  "⏳ Poyo 正在工作...",
	StatusThinking: "🌀 Poyo 正在思考...",
	StatusError:    "❌ 出错了",

	InputPrompt: "💬 你: ",
	InputHelp:   "按 Enter 发送，Ctrl+C 退出",

	PanelTools:   "🔧 能力",
	PanelFiles:   "📂 文件",
	PanelHistory: "📜 历史",

	Shortcuts: `
⌨️ 快捷键:
  Ctrl+C     - 取消当前操作
  Ctrl+D     - 退出 Poyo
  Ctrl+L     - 清屏
  ↑/↓        - 历史命令
  Tab        - 自动补全
`,
}

// CLI related prompts
var CLI = struct {
	// Command descriptions
	CmdRoot    string
	CmdAbility string
	CmdPlugin  string
	CmdSession string

	// Flag descriptions
	FlagModel      string
	FlagPermission string
	FlagDebug      string
	FlagSession    string
	FlagConfig     string
}{
	CmdRoot: `💚 Poyo - 星之卡比风格的智能代码助手

就像星之卡比能吸入并复制敌人的能力一样，Poyo 可以：
  • 🌀 吸入并理解各种代码库
  • ⭐ 复制并应用不同的编程模式
  • 💪 使用各种"能力"(Tools) 来完成任务
  • 🌙 在梦之国 (Dream Land) 中自由探索

Poyo 支持 Lua、MCP、Python、Node.js 等多种插件类型，
让你可以扩展它的能力！

示例:
  poyo "帮我实现一个 REST API"
  poyo -i                    # 交互模式
  poyo -m claude-opus-4-6    # 指定模型
  poyo -p accept-edits       # 自动接受编辑
  poyo --list-sessions       # 列出所有会话`,

	CmdPlugin: `🔌 管理和查看 Poyo 的插件

Poyo 支持多种插件类型：
  • Lua Plugin   - 原生 Poyo 插件，访问完整 API
  • MCP Plugin   - Model Context Protocol 插件
  • Script Plugin - Shell/Python/Node.js 脚本插件`,

	CmdAbility: `💚 管理和查看 Poyo 的能力 (Tools)

每个能力都对应星之卡比的一个 Copy Ability：
  ⚔️  Sword   - 代码编辑和修改
  🔥  Fire    - 执行命令
  🧊  Ice     - 文件搜索
  ⚡  Spark   - 快速读取
  🔆  Beam    - 网络请求
  🪨  Stone   - 稳定的写入操作
  🪃  Cutter  - 精确的编辑
  🥷  Ninja   - 隐秘的代理操作`,

	CmdSession: `📝 管理会话

Poyo 会自动保存对话历史，你可以随时恢复之前的会话。`,

	FlagModel:      "使用的模型 (如 claude-opus-4-6, claude-sonnet-4-6)",
	FlagPermission: "权限模式",
	FlagDebug:      "启用调试日志",
	FlagSession:    "恢复指定会话",
	FlagConfig:     "配置文件路径",
}

// GetWelcomeMessage returns a random welcome message
func GetWelcomeMessage() string {
	return TUI.Welcome
}

// GetHelpText returns the help text
func GetHelpText() string {
	return TUI.HelpText
}

// GetStatusMessage returns a status message
func GetStatusMessage(status string) string {
	switch status {
	case "ready":
		return TUI.StatusReady
	case "working":
		return TUI.StatusWorking
	case "thinking":
		return TUI.StatusThinking
	case "error":
		return TUI.StatusError
	default:
		return status
	}
}
