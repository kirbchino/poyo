# 💚 Poyo 工具系统 - 完整能力列表

## 🔤 POYO 解构

```
P = Portal      门户 — 所有 Agent 的统一入口
O = Orchestrator 编排器 — 协调插件、能力、工具
Y = Yield       产出 — 生成结果、交付价值
O = Omnibus     包罗万象 — 兼容所有插件格式
```

## 🌀 卡比能力系统

每个工具都对应星之卡比的一个 Copy Ability！

| 工具 | 卡比能力 | 图标 | 功能 |
|------|---------|------|------|
| **Read** | Inhale | 🌀 | 吸入文件内容，支持代码、配置、图片、PDF |
| **Write** | Stone | 💪 | 石头写入，稳定可靠的文件创建 |
| **Edit** | Sword | ⚔️ | 剑士编辑，精准修改文件内容 |
| **Glob** | Cutter | 🪃 | 刀片搜索，快速匹配文件模式 |
| **Grep** | Spark | ⚡ | 闪电搜索，快速搜索文件内容 |
| **Bash** | Fire | 🔥 | 火焰执行，执行 Shell 命令 |
| **Agent** | Ninja | 🥷 | 忍者分身，创建子代理执行任务 |
| **TodoWrite** | Todo | 📝 | 任务追踪，管理待办事项 |
| **TaskOutput** | Output | 📤 | 任务结果，获取后台任务输出 |
| **TaskStop** | Stop | 🛑 | 任务停止，停止后台任务 |
| **WebFetch** | Beam | 🌐 | 光束获取，从网络获取内容 |
| **WebSearch** | Search | 🔎 | 光束搜索，互联网搜索（支持 Brave/Google API） |
| **NotebookEdit** | Notebook | 📓 | 笔记本编辑，处理 Jupyter Notebook |
| **AskUserQuestion** | Ask | 💬 | 用户交互，向用户提问 |
| **EnterPlanMode** | Plan | 📋 | 规划模式，进入规划状态 |
| **ExitPlanMode** | Exit | ✅ | 规划完成，退出规划状态 |
| **CronCreate** | TimeWarp | ⏰ | 时间扭曲，创建定时任务 |
| **CronDelete** | TimeCancel | 🗑️ | 时间取消，删除定时任务 |
| **CronList** | TimeList | 📅 | 时间列表，查看定时任务 |
| **EnterWorktree** | Leaf | 🌿 | 叶子分身，创建 Git Worktree |
| **ExitWorktree** | Return | 🚪 | 回归本源，退出 Git Worktree |
| **Skill** | Copy | ⭐ | 技能复制，调用预定义技能 |
| **MediaRead** | InhaleMedia | 📸 | 媒体吸入，读取图片和 PDF |

## 🔧 工具详细说明

### 🌀 文件操作

#### Read - 吸入
```lua
poyo.use("Read", {file_path = "/path/to/file", offset = 1, limit = 100})
```
- 读取文件内容
- 支持分页读取
- 支持图片/PDF（通过 MediaRead）

#### Write - 石头写入
```lua
poyo.use("Write", {file_path = "/path/to/file", content = "Hello!"})
```
- 创建或覆盖文件
- 自动创建父目录

#### Edit - 剑士编辑
```lua
poyo.use("Edit", {
    file_path = "/path/to/file",
    old_string = "old",
    new_string = "new"
})
```
- 精准替换文件内容
- 不伤及无关代码

#### Glob - 刀片搜索
```lua
poyo.use("Glob", {pattern = "**/*.go"})
```
- 通配符模式匹配
- 快速定位文件

#### Grep - 闪电搜索
```lua
poyo.use("Grep", {
    pattern = "func main",
    path = ".",
    type = "go"
})
```
- 正则表达式搜索
- 支持文件类型过滤

### 🔥 执行

#### Bash - 火焰执行
```lua
poyo.use("Bash", {
    command = "go build ./...",
    description = "Build the project",
    timeout = 60000
})
```
- 执行 Shell 命令
- 危险命令检测
- 支持后台执行

#### Agent - 忍者分身
```lua
poyo.use("Agent", {
    prompt = "分析这个代码库的结构",
    subagent_type = "explore",
    max_turns = 10,
    run_in_background = true
})
```
- 创建子代理
- 支持 explore/plan 类型
- 后台执行支持

### 🌐 网络

#### WebFetch - 光束获取
```lua
poyo.use("WebFetch", {
    url = "https://example.com",
    method = "GET"
})
```

#### WebSearch - 光束搜索
```lua
poyo.use("WebSearch", {
    query = "Go best practices",
    count = 10,
    fresh = "week"
})
```
- 支持 Brave Search API
- 支持 Google Custom Search
- 时效性过滤

### 🌿 Git Worktree

#### EnterWorktree - 叶子分身
```lua
poyo.use("EnterWorktree", {name = "feature-branch"})
```
- 创建隔离的工作区
- 自动创建新分支

#### ExitWorktree - 回归本源
```lua
poyo.use("ExitWorktree", {action = "remove", discard_changes = false})
```
- 退出 Worktree
- 可选择保留或删除

### ⭐ Skills

#### Skill - 技能复制
```lua
poyo.use("Skill", {skill = "pdf", args = "document.pdf"})
```
- 调用预定义技能
- 类似 CC 的 Skill 系统

### 📸 媒体

#### MediaRead - 媒体吸入
```lua
poyo.use("MediaRead", {
    path = "image.png",
    type = "auto"  -- image, pdf, auto
})
```
- 图片读取（PNG, JPG, GIF, WebP, BMP）
- PDF 读取

### 📋 任务管理

#### TodoWrite - 任务追踪
```lua
poyo.use("TodoWrite", {
    todos = {
        {content = "实现功能A", status = "in_progress", activeForm = "实现功能A中"},
        {content = "测试功能B", status = "pending", activeForm = "测试功能B中"}
    }
})
```

#### TaskOutput - 任务结果
```lua
poyo.use("TaskOutput", {task_id = "agent_xxx", block = true, timeout = 30000})
```

#### TaskStop - 任务停止
```lua
poyo.use("TaskStop", {task_id = "agent_xxx"})
```

### ⏰ 定时任务

#### CronCreate
```lua
poyo.use("CronCreate", {
    cron = "0 9 * * *",
    prompt = "每天早上提醒我查看邮件"
})
```

#### CronDelete
```lua
poyo.use("CronDelete", {id = "cron_xxx"})
```

#### CronList
```lua
poyo.use("CronList", {})
```

### 💬 用户交互

#### AskUserQuestion
```lua
poyo.use("AskUserQuestion", {
    question = "选择一个选项",
    options = {"A", "B", "C"}
})
```

### 📋 规划模式

#### EnterPlanMode
```lua
poyo.use("EnterPlanMode", {task_description = "设计用户认证系统"})
```

#### ExitPlanMode
```lua
poyo.use("ExitPlanMode", {})
```

---

## 🔌 MCP 工具

MCP 插件的工具会动态注册到工具列表中，使用方式与内置工具相同：

```lua
-- 假设 MCP 插件 "weather" 提供了 "get_weather" 工具
poyo.use("mcp_weather_get_weather", {city = "Beijing"})
```

---

💚 **Poyo~! 所有能力已就绪！** ☆彡
