package prompt

// Hook-related prompts and messages
var Hooks = struct {
	// Hook type descriptions
	PreToolUse  string
	PostToolUse string
	PrePrompt   string
	PostPrompt  string
	OnStart     string
	OnEnd       string
	OnError     string

	// Hook execution messages
	HookExecuting string
	HookBlocked   string
	HookModified  string
}{
	PreToolUse: `🗡️ **PreToolUse Hook** - 工具使用前

在 Poyo 使用任何能力之前触发的钩子。
可以：
- 阻止工具执行
- 修改工具输入
- 记录工具使用

返回格式：
{
  "blocked": false,    // 是否阻止执行
  "modified": false,   // 是否修改了输入
  "input": {...}       // 修改后的输入（如果 modified=true）
}`,

	PostToolUse: `✅ **PostToolUse Hook** - 工具使用后

在 Poyo 使用能力完成后触发的钩子。
可以：
- 修改工具输出
- 记录执行结果
- 触发后续操作

返回格式：
{
  "modified": false,   // 是否修改了输出
  "result": {...}      // 修改后的结果（如果 modified=true）
}`,

	PrePrompt: `📝 **PrePrompt Hook** - 提示处理前

在 Poyo 处理用户输入之前触发的钩子。
可以修改或增强用户提示。

返回格式：
{
  "modified": false,
  "prompt": "..."     // 修改后的提示
}`,

	PostPrompt: `📤 **PostPrompt Hook** - 提示处理后

在 Poyo 生成响应之后触发的钩子。
可以修改最终输出。

返回格式：
{
  "modified": false,
  "response": "..."   // 修改后的响应
}`,

	OnStart: `🌟 **OnStart Hook** - 会话开始

Poyo 开始新会话时触发。
适合做初始化工作。

返回格式：
{
  "message": "欢迎消息（可选）"
}`,

	OnEnd: `🌙 **OnEnd Hook** - 会话结束

Poyo 结束会话时触发。
适合做清理工作。

返回格式：
{
  "message": "告别消息（可选）"
}`,

	OnError: `⚠️ **OnError Hook** - 错误处理

Poyo 遇到错误时触发。
可以记录错误或尝试恢复。

返回格式：
{
  "handled": false,   // 是否已处理
  "message": "..."    // 自定义错误消息
}`,

	HookExecuting: "🔗 Poyo 正在执行钩子：{{.hook}}",
	HookBlocked:   "🚫 钩子阻止了操作：{{.reason}}",
	HookModified:  "✏️ 钩子修改了内容",
}

// Plugin-related prompts
var Plugin = struct {
	// Plugin type descriptions
	LuaPlugin    string
	MCPPlugin    string
	ScriptPlugin string

	// Plugin status messages
	PluginLoading  string
	PluginLoaded   string
	PluginError    string
	PluginUnloaded string
}{
	LuaPlugin: `💚 **Lua Plugin** - 原生 Poyo 插件

使用 Lua 脚本编写的 Poyo 原生插件。
可以访问完整的 poyo.* API。

manifest.json:
{
  "id": "my-plugin",
  "name": "My Plugin",
  "version": "1.0.0",
  "type": "lua",
  "main": "main.lua"
}`,

	MCPPlugin: `🔌 **MCP Plugin** - Model Context Protocol

通过 MCP 协议连接的外部工具。
支持 Tools、Resources、Prompts 等。

manifest.json:
{
  "id": "my-mcp",
  "name": "My MCP Server",
  "version": "1.0.0",
  "type": "mcp",
  "main": "node server.js"
}`,

	ScriptPlugin: `📜 **Script Plugin** - 脚本插件

通过 Shell/Python/Node.js 脚本实现的插件。
适合快速集成现有工具。

manifest.json:
{
  "id": "my-script",
  "name": "My Script",
  "version": "1.0.0",
  "type": "script",
  "main": "script.sh"
}`,

	PluginLoading:  "🔄 Poyo 正在加载插件：{{.name}}",
	PluginLoaded:   "✅ Poyo 加载了新能力：{{.name}}",
	PluginError:    "❌ 插件加载失败：{{.name}} - {{.error}}",
	PluginUnloaded: "👋 Poyo 卸载了插件：{{.name}}",
}

// GetHookDescription returns the description for a hook type
func GetHookDescription(hookType string) string {
	switch hookType {
	case "PreToolUse":
		return Hooks.PreToolUse
	case "PostToolUse":
		return Hooks.PostToolUse
	case "PrePrompt":
		return Hooks.PrePrompt
	case "PostPrompt":
		return Hooks.PostPrompt
	case "OnStart":
		return Hooks.OnStart
	case "OnEnd":
		return Hooks.OnEnd
	case "OnError":
		return Hooks.OnError
	default:
		return hookType + " - Poyo 的一个钩子"
	}
}

// GetPluginTypeDescription returns the description for a plugin type
func GetPluginTypeDescription(pluginType string) string {
	switch pluginType {
	case "lua":
		return Plugin.LuaPlugin
	case "mcp":
		return Plugin.MCPPlugin
	case "script":
		return Plugin.ScriptPlugin
	default:
		return pluginType + " - Poyo 的一个插件类型"
	}
}
