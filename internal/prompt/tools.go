package prompt

// Tool descriptions with Kirby-themed personality
// Each tool is presented as a "Copy Ability" that Poyo can use

// ToolDescriptions contains all tool descriptions
var ToolDescriptions = struct {
	// File operations
	Read   string
	Write  string
	Edit   string
	Glob   string
	Grep   string

	// Execution
	Bash  string
	Agent string

	// User interaction
	AskUserQuestion string
	EnterPlanMode   string
	ExitPlanMode    string

	// Task management
	TodoWrite  string
	TaskOutput string
	TaskStop   string

	// Scheduling
	CronCreate string
	CronDelete string
	CronList   string

	// Network
	WebFetch  string
	WebSearch string

	// Notebook
	NotebookEdit string

	// Skills
	Skill string

	// Worktree
	EnterWorktree string
	ExitWorktree  string

	// Media
	MediaRead string
}{
	Read: `🌀 **Inhale Read** - 吸入文件内容

Poyo 的吸入能力！读取文件并理解其内容。
就像卡比吸入食物一样，可以轻松消化各种文件格式：
- 代码文件（高亮显示）
- 配置文件（JSON、YAML、TOML）
- 图片文件（PNG、JPG）
- PDF 文档

用法：指定文件路径，Poyo 会吸入并展示内容。`,

	Write: `💪 **Stone Write** - 石头写入

石头卡比的能力！稳定可靠地写入文件。
就像石头形态一样坚固，确保文件安全保存。

用法：指定路径和内容，Poyo 会创建或覆盖文件。`,

	Edit: `⚔️ **Sword Edit** - 剑士编辑

剑士卡比的精确斩击！精准修改文件中的特定内容。
每一刀都精确命中目标，不伤及无辜代码。

用法：指定文件、旧内容和新内容，Poyo 精准替换。`,

	Glob: `🔍 **Cutter Glob** - 刀片搜索

刀片卡比的回旋镖！快速搜索匹配模式的文件。
就像回旋镖一样，精准命中目标后返回。

用法：使用通配符模式搜索文件。`,

	Grep: `⚡ **Spark Grep** - 闪电搜索

闪电卡比的速度！在文件内容中快速搜索匹配的文本。
电光火石间找到你需要的代码。

用法：使用正则表达式搜索文件内容。`,

	Bash: `🔥 **Fire Bash** - 火焰执行

火焰卡比的能量！执行 Shell 命令完成任务。
强大但需要小心使用，避免误伤。

用法：执行 Shell 命令，获取输出结果。`,

	Agent: `🎭 **Ninja Agent** - 忍者分身

忍者卡比的分身术！创建子代理独立执行任务。
分身会在后台默默工作，完成后汇报结果。

用法：创建专门的子代理处理复杂任务。`,

	AskUserQuestion: `💬 **Poyo Ask** - Poyo 询问

Poyo 直接与用户对话！
当遇到不确定的情况，Poyo 会礼貌地询问。

用法：向用户提问并等待回答。`,

	EnterPlanMode: `📋 **Plan Mode** - 规划模式

进入规划模式，仔细分析任务再执行。
就像卡比观察敌人后再行动一样谨慎。

用法：进入规划模式，制定详细计划。`,

	ExitPlanMode: `✅ **Plan Complete** - 规划完成

退出规划模式，开始执行计划。
Poyo 准备好了，出发！

用法：退出规划模式，开始执行。`,

	TodoWrite: `📝 **Todo Track** - 任务追踪

追踪任务进度，确保不遗漏任何步骤。
Poyo 会记住所有待办事项。

用法：更新任务列表和状态。`,

	TaskOutput: `📤 **Task Result** - 任务结果

获取后台任务的输出结果。
查看分身们的工作成果。

用法：获取指定任务的输出。`,

	TaskStop: `🛑 **Task Stop** - 任务停止

停止正在运行的后台任务。
当任务不再需要时，及时终止。

用法：停止指定的后台任务。`,

	CronCreate: `⏰ **Time Warp** - 时间扭曲

创建定时任务，在未来某个时间执行。
Poyo 会记住并在指定时间触发。

用法：设置定时任务。`,

	CronDelete: `🗑️ **Time Cancel** - 时间取消

取消已设置的定时任务。

用法：删除指定的定时任务。`,

	CronList: `📅 **Time List** - 时间列表

列出所有已设置的定时任务。

用法：查看所有定时任务。`,

	WebFetch: `🌐 **Beam Fetch** - 光束获取

光束卡比的能力！从网络获取内容。
就像光束一样，快速准确地获取远程资源。

用法：从 URL 获取网页内容。`,

	WebSearch: `🔎 **Beam Search** - 光束搜索

在互联网上搜索信息。
光束扫描全网，找到你需要的内容。

用法：搜索互联网信息。`,

	NotebookEdit: `📓 **Notebook Edit** - 笔记本编辑

编辑 Jupyter Notebook 文件。
处理数据分析和科学计算任务。

用法：编辑 .ipynb 文件。`,

	Skill: `⭐ **Copy Skill** - 技能复制

复制并使用预定义的技能！
就像卡比复制敌人的特殊能力一样强大。
Skills 是更强大、更专业的功能模块。

用法：调用 /skill-name 格式的技能。`,

	EnterWorktree: `🌿 **Leaf Worktree** - 叶子分身

创建 Git Worktree 隔离工作区！
就像卡比在梦之国的不同区域探索。
在独立分支上安全工作，不影响主分支。

用法：创建并进入新的 Git Worktree。`,

	ExitWorktree: `🚪 **Return Home** - 回归本源

退出 Git Worktree，返回主工作区。
可选择保留或删除 Worktree。

用法：退出当前 Worktree。`,

	MediaRead: `📸 **Inhale Media** - 媒体吸入

吸入图片和 PDF 文件！
就像卡比吞噬一切一样，图片和文档都能理解。

支持格式：
- 图片：PNG, JPG, GIF, WebP, BMP
- 文档：PDF

用法：读取图片或 PDF 文件内容。`,
}

// GetToolDescription returns the Kirby-styled description for a tool
func GetToolDescription(name string) string {
	switch name {
	case "Read":
		return ToolDescriptions.Read
	case "Write":
		return ToolDescriptions.Write
	case "Edit":
		return ToolDescriptions.Edit
	case "Glob":
		return ToolDescriptions.Glob
	case "Grep":
		return ToolDescriptions.Grep
	case "Bash":
		return ToolDescriptions.Bash
	case "Agent":
		return ToolDescriptions.Agent
	case "AskUserQuestion":
		return ToolDescriptions.AskUserQuestion
	case "EnterPlanMode":
		return ToolDescriptions.EnterPlanMode
	case "ExitPlanMode":
		return ToolDescriptions.ExitPlanMode
	case "TodoWrite":
		return ToolDescriptions.TodoWrite
	case "TaskOutput":
		return ToolDescriptions.TaskOutput
	case "TaskStop":
		return ToolDescriptions.TaskStop
	case "CronCreate":
		return ToolDescriptions.CronCreate
	case "CronDelete":
		return ToolDescriptions.CronDelete
	case "CronList":
		return ToolDescriptions.CronList
	case "WebFetch":
		return ToolDescriptions.WebFetch
	case "WebSearch":
		return ToolDescriptions.WebSearch
	case "NotebookEdit":
		return ToolDescriptions.NotebookEdit
	case "Skill":
		return ToolDescriptions.Skill
	case "EnterWorktree":
		return ToolDescriptions.EnterWorktree
	case "ExitWorktree":
		return ToolDescriptions.ExitWorktree
	case "MediaRead":
		return ToolDescriptions.MediaRead
	default:
		return name + " - Poyo 的一个能力"
	}
}
