package prompt

import "fmt"

// Message templates with Kirby-style personality
var Messages = struct {
	// Permission messages
	PermissionAsk    string
	PermissionDenied string
	PermissionAuto   string

	// Error messages
	ErrorGeneric      string
	ErrorToolNotFound string
	ErrorTimeout      string
	ErrorContext      string

	// Progress messages
	ProgressStart   string
	ProgressWorking string
	ProgressDone    string

	// Session messages
	SessionNew    string
	SessionResume string
	SessionSaved  string

	// Tool-specific messages
	ToolBashDangerous string
	ToolBashRunning   string
	ToolFileNotFound  string
	ToolFileLarge     string
}{
	PermissionAsk: "🔐 Poyo 需要你的许可：{{.action}}",
	PermissionDenied: "🚫 Poyo 没有权限执行此操作：{{.action}}",
	PermissionAuto: "✅ Poyo 自动获得权限：{{.action}}",

	ErrorGeneric:      "😅 哎呀，Poyo 遇到了问题：{{.error}}",
	ErrorToolNotFound: "🔍 Poyo 找不到这个能力：{{.tool}}",
	ErrorTimeout:      "⏰ Poyo 等待超时了...",
	ErrorContext:      "💭 Poyo 的上下文好像有点问题...",

	ProgressStart:   "🌀 Poyo 开始工作啦！",
	ProgressWorking: "⏳ Poyo 正在努力处理中...",
	ProgressDone:    "💚 Poyo 完成啦！",

	SessionNew:    "🌟 新的冒险开始！Poyo 准备就绪！",
	SessionResume: "🔄 Poyo 回来了！继续上次的冒险！",
	SessionSaved:  "💾 Poyo 记住了这次的旅程！",

	ToolBashDangerous: "⚠️ 这个命令看起来有点危险，Poyo 需要确认一下：{{.command}}",
	ToolBashRunning:   "🔥 正在执行：{{.command}}",
	ToolFileNotFound:  "📂 Poyo 找不到这个文件：{{.path}}",
	ToolFileLarge:     "📦 这个文件有点大呢（{{.size}}），Poyo 正在努力读取...",
}

// FormatMessage formats a message template with variables
func FormatMessage(template string, vars map[string]string) string {
	result := template
	for k, v := range vars {
		result = replaceAll(result, "{{."+k+"}}", v)
	}
	return result
}

// Simple helper to replace placeholders
func replaceAll(s, old, new string) string {
	// Simple implementation without importing strings
	result := ""
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

// Quick message generators for common cases
func MsgPermissionAsk(action string) string {
	return fmt.Sprintf("🔐 Poyo 需要你的许可：%s", action)
}

func MsgErrorGeneric(err string) string {
	return fmt.Sprintf("😅 哎呀，Poyo 遇到了问题：%s", err)
}

func MsgToolNotFound(tool string) string {
	return fmt.Sprintf("🔍 Poyo 找不到这个能力：%s", tool)
}

func MsgProgressStart() string {
	return "🌀 Poyo 开始工作啦！"
}

func MsgProgressDone() string {
	return "💚 Poyo 完成啦！"
}

func MsgSessionNew() string {
	return "🌟 新的冒险开始！Poyo 准备就绪！"
}

func MsgSessionResume() string {
	return "🔄 Poyo 回来了！继续上次的冒险！"
}

func MsgFileNotFound(path string) string {
	return fmt.Sprintf("📂 Poyo 找不到这个文件：%s", path)
}

func MsgBashDangerous(cmd string) string {
	return fmt.Sprintf("⚠️ 这个命令看起来有点危险，Poyo 需要确认一下：%s", cmd)
}

// Kirby-style fun messages
func PoyoGreeting() string {
	return "💚 Poyo~! 欢迎来到梦之国！有什么可以帮你的吗？"
}

func PoyoSuccess() string {
	return "⭐ 能力释放成功！Poyo 超开心！"
}

func PoyoThinking() string {
	return "🌀 Poyo 正在吸入信息..."
}

func PoyoCelebration() string {
	return "🎉 Poyo~! 太棒了！任务圆满完成！"
}

func PoyoDance() string {
	return "💃 Poyo 跳舞庆祝！☆彡"
}
