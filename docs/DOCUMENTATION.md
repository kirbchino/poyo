# 💚 Poyo 技术文档

> **Portal Of Your Orchestrated Omnibus-agents**
>
> 编排你所有 Agent 的统一门户平台

---

## 目录

1. [项目概述](#项目概述)
2. [核心架构](#核心架构)
3. [功能模块](#功能模块)
4. [快速开始](#快速开始)
5. [配置指南](#配置指南)
6. [API 参考](#api-参考)
7. [插件系统](#插件系统)
8. [多 Agent 协作](#多-agent-协作)
9. [测试指南](#测试指南)
10. [开发指南](#开发指南)

---

## 项目概述

### 什么是 Poyo？

Poyo 是一个星之卡比风格的智能代码助手框架，使用 Go 语言实现。就像星之卡比能吸入并复制敌人的能力一样，Poyo 可以：

- 🌀 **Inhale 吸入** — 摄取代码库、文档、知识
- ⭐ **Copy 复制** — 学习模式并应用到新场景
- 💪 **Ability 能力** — 使用工具完成各种任务
- 🌙 **Dream Land 梦之国** — 在统一环境中自由探索
- 🔌 **Omnibus 包容** — 兼容 CC、OpenClaw、MCP 等多种插件格式

### 设计理念

```
┌─────────────────────────────────────────────────────────────┐
│                      Poyo 核心理念                           │
├─────────────────────────────────────────────────────────────┤
│  1. 统一门户 - 所有 Agent 能力的统一入口                      │
│  2. 能力复用 - 像卡比一样吸收并复用各种能力                    │
│  3. 插件生态 - 兼容多种插件格式，拥抱开放生态                  │
│  4. 安全可控 - 完善的权限系统和沙箱隔离                       │
│  5. 可扩展性 - Hook 机制支持自定义扩展                        │
└─────────────────────────────────────────────────────────────┘
```

### 项目统计

| 指标 | 数值 |
|------|------|
| 核心代码 | ~42,000 行 Go |
| 内置工具 | 25+ |
| 命令系统 | 109 个斜杠命令 |
| Hook 事件 | 26 种 |
| 测试用例 | 84+ 个 |
| 测试通过率 | 100% |

---

## 核心架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    cmd/poyo/main.go                      │    │
│  │         (Cobra CLI, Flags, Subcommands)                  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        TUI Layer                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   internal/tui/                          │    │
│  │         (Bubble Tea, Input, Messages, Streaming)         │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Query Engine                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                 internal/query/engine.go                 │    │
│  │           (Message Processing, Tool Routing)             │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  Tools Layer  │     │  Plugin Layer │     │   MCP Layer   │
│ internal/tools│     │internal/plugin│     │ internal/mcp  │
│               │     │               │     │               │
│ • Read        │     │ • Lua VM      │     │ • Client      │
│ • Write       │     │ • WASM        │     │ • Manager     │
│ • Edit        │     │ • Script      │     │ • Protocol    │
│ • Bash        │     │ • Compat      │     │ • OAuth       │
│ • Glob        │     │ • HotReload   │     │               │
│ • Grep        │     │               │     │               │
│ • Agent       │     │               │     │               │
│ • WebFetch    │     │               │     │               │
│ • WebSearch   │     │               │     │               │
│ • ...         │     │               │     │               │
└───────────────┘     └───────────────┘     └───────────────┘
```

### 目录结构

```
poyo/
├── cmd/poyo/                    # CLI 入口
│   ├── main.go                  # 主程序
│   └── main_test.go             # 入口测试
│
├── internal/                    # 内部模块
│   ├── agent/                   # 子代理系统
│   │   ├── executor.go          # Agent 执行器
│   │   ├── tool.go              # Agent 工具接口
│   │   └── types.go             # Agent 类型定义
│   │
│   ├── brief/                   # 报告生成
│   │   └── brief.go             # 简报生成器
│   │
│   ├── commands/                # 命令系统
│   │   ├── builtin.go           # 内置命令
│   │   ├── builtin_extended.go  # 扩展命令
│   │   ├── executor.go          # 命令执行器
│   │   └── registry.go          # 命令注册表
│   │
│   ├── compactor/               # 会话压缩
│   │   └── compactor.go         # 压缩策略实现
│   │
│   ├── computeruse/             # 计算机使用
│   │   └── computer.go          # 计算机控制
│   │
│   ├── config/                  # 配置管理
│   │   └── config.go            # 配置加载
│   │
│   ├── filehistory/             # 文件历史
│   │   └── history.go           # 历史管理
│   │
│   ├── hooks/                   # Hook 系统
│   │   ├── config.go            # Hook 配置
│   │   ├── executor.go          # Hook 执行器
│   │   ├── http_hook.go         # HTTP Hook
│   │   └── types.go             # Hook 类型
│   │
│   ├── interaction/             # 用户交互
│   │   ├── tool.go              # 交互工具
│   │   └── types.go             # 交互类型
│   │
│   ├── lsp/                     # LSP 工具
│   │   ├── tool.go              # LSP 工具实现
│   │   └── types.go             # LSP 类型
│   │
│   ├── mcp/                     # MCP 集成
│   │   ├── client.go            # MCP 客户端
│   │   ├── manager.go           # MCP 管理器
│   │   ├── oauth.go             # OAuth 认证
│   │   └── protocol.go          # MCP 协议
│   │
│   ├── notebook/                # Notebook 支持
│   │   ├── notebook.go          # Notebook 管理
│   │   └── tool.go              # Notebook 工具
│   │
│   ├── permission/              # 权限系统
│   │   ├── classifier.go        # 权限分类
│   │   ├── manager.go           # 权限管理
│   │   └── types.go             # 权限类型
│   │
│   ├── prompt/                  # Prompt 系统
│   │   ├── prompts.go           # 提示词定义
│   │   ├── system.go            # 系统提示
│   │   └── tools.go             # 工具描述
│   │
│   ├── query/                   # 查询引擎
│   │   ├── engine.go            # 查询引擎
│   │   └── executor.go          # 查询执行
│   │
│   ├── sandbox/                 # 沙箱隔离
│   │   └── sandbox.go           # 沙箱实现
│   │
│   ├── services/                # 服务层
│   │   ├── api/                 # API 客户端
│   │   ├── audit/               # 审计日志
│   │   ├── plugin/              # 插件服务
│   │   └── streaming/           # 流式处理
│   │
│   ├── session/                 # 会话管理
│   │   └── manager.go           # 会话管理器
│   │
│   ├── task/                    # 任务管理
│   │   ├── manager.go           # 任务管理器
│   │   └── bash_executor.go     # Bash 执行器
│   │
│   ├── team/                    # 多 Agent 团队
│   │   └── team.go              # 团队管理
│   │
│   ├── tools/                   # 核心工具
│   │   ├── registry.go          # 工具注册表
│   │   ├── tool.go              # 工具接口
│   │   ├── bash.go              # Bash 工具
│   │   ├── fileread.go          # 文件读取
│   │   ├── filewrite.go         # 文件写入
│   │   ├── fileedit.go          # 文件编辑
│   │   ├── glob.go              # 文件搜索
│   │   ├── grep.go              # 内容搜索
│   │   ├── agent.go             # 子代理
│   │   ├── webfetch.go          # 网络获取
│   │   ├── websearch.go         # 网络搜索
│   │   └── ...                  # 其他工具
│   │
│   ├── tui/                     # 终端 UI
│   │   ├── app.go               # 应用入口
│   │   ├── input.go             # 输入处理
│   │   ├── messages.go          # 消息显示
│   │   ├── model.go             # 数据模型
│   │   └── streaming.go         # 流式渲染
│   │
│   ├── types/                   # 类型定义
│   │   ├── message.go           # 消息类型
│   │   └── permission.go        # 权限类型
│   │
│   ├── utils/                   # 工具函数
│   │   └── tokenizer/           # Token 计数
│   │
│   └── vim/                     # Vim 模式
│       └── vim.go               # Vim 实现
│
├── plugins/                     # 插件示例
│   ├── poyo-official/           # 官方插件
│   │   ├── memory/              # 记忆插件
│   │   └── git-helper/          # Git 辅助
│   ├── lua-example/             # Lua 示例
│   ├── wasm-example/            # WASM 示例
│   └── script-python/           # Python 脚本
│
├── tests/                       # 端到端测试
│   └── e2e/                     # E2E 测试
│
└── docs/                        # 文档
```

---

## 功能模块

### 1. 工具系统

Poyo 内置 25+ 工具，覆盖文件操作、代码编辑、网络请求等场景：

| 工具 | 能力 | 说明 |
|------|------|------|
| Read | 🌀 Inhale | 吸入文件内容 |
| Write | 💪 Stone | 石头写入 |
| Edit | ⚔️ Sword | 剑士编辑 |
| Glob | 🪃 Cutter | 刀片搜索 |
| Grep | ⚡ Spark | 闪电搜索 |
| Bash | 🔥 Fire | 火焰执行 |
| Agent | 🥷 Ninja | 忍者分身 |
| WebFetch | 🌐 Beam | 光束获取 |
| WebSearch | 🔎 Search | 网络搜索 |
| Skill | ⭐ Copy | 技能复制 |
| NotebookEdit | 📓 Notebook | 笔记编辑 |

### 2. Hook 系统

支持 26 种 Hook 事件类型：

```go
// Hook 事件类型
const (
    PreToolUse    // 工具使用前
    PostToolUse   // 工具使用后
    SessionStart  // 会话开始
    SessionEnd    // 会话结束
    PrePrompt     // 提示前
    PostPrompt    // 提示后
    Notification  // 通知
    Stop          // 停止
    // ... 更多事件
)
```

Hook 配置示例：

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "type": "command",
        "command": "echo 'Tool about to be used: {{tool_name}}'"
      }
    ],
    "PostToolUse": [
      {
        "type": "http",
        "url": "https://api.example.com/log",
        "method": "POST"
      }
    ]
  }
}
```

### 3. 权限系统

支持 5 种权限模式：

| 模式 | 说明 |
|------|------|
| `ask` | 每次操作询问用户 |
| `auto` | 自动处理常见操作 |
| `accept-all` | 接受所有操作 |
| `accept-edits` | 接受编辑操作 |
| `bypass` | 跳过权限检查 |

### 4. 命令系统

提供 109 个斜杠命令，分类管理：

```
/session     # 会话管理
  /help      # 帮助
  /clear     # 清屏
  /compact   # 压缩
  /history   # 历史
  /save      # 保存
  /load      # 加载

/config      # 配置管理
  /model     # 模型切换
  /theme     # 主题设置
  /set       # 设置变量
  /get       # 获取变量

/git         # Git 操作
  /commit    # 提交
  /review    # 评审
  /branch    # 分支
  /checkout  # 切换

/file        # 文件操作
  /ls        # 列表
  /tree      # 树形
  /read      # 读取
  /edit      # 编辑
```

### 5. 上下文压缩

支持 4 种压缩策略：

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| `truncate` | 截断旧消息 | 快速压缩 |
| `summarize` | 生成摘要 | 保留语义 |
| `semantic` | 语义压缩 | 智能筛选 |
| `hierarchical` | 分层压缩 | 复杂对话 |

配置示例：

```go
config := CompressionConfig{
    MaxTokens:     100000,    // 最大 Token 数
    PreserveRecent: 5,         // 保留最近消息数
    Strategy:      "semantic", // 压缩策略
}
```

---

## 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/kirbchino/poyo.git
cd poyo

# 安装依赖
go mod download

# 构建
go build -o poyo ./cmd/poyo

# 安装到系统
sudo cp poyo /usr/local/bin/
```

### 基本使用

```bash
# 单次查询
poyo "帮我实现一个 REST API"

# 交互模式
poyo -i

# 指定模型
poyo -m claude-opus-4-6 "分析这个代码"

# 指定权限模式
poyo -p auto "运行测试"

# 加载会话
poyo -s session-abc123

# 查看会话列表
poyo --list-sessions
```

### 配置文件

创建 `~/.poyo/config.json`：

```json
{
  "model": "claude-sonnet-4-6",
  "permission_mode": "ask",
  "api": {
    "base_url": "https://api.anthropic.com",
    "api_key": "your-api-key",
    "type": "anthropic"
  },
  "hooks": {
    "PreToolUse": []
  },
  "max_tokens": 100000,
  "debug": false
}
```

---

## 配置指南

### API 配置

支持多种 API 类型：

```json
{
  "api": {
    "type": "anthropic",  // anthropic, openai, custom
    "base_url": "https://api.anthropic.com",
    "api_key": "sk-xxx",
    "custom_headers": {
      "X-Custom-Header": "value"
    }
  }
}
```

### 权限配置

权限规则来源（优先级从高到低）：

1. 命令行参数 `-p`
2. 环境变量 `POYO_PERMISSION_MODE`
3. 配置文件 `permission_mode`
4. 默认值 `ask`

### Hook 配置

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "type": "command",
        "command": "/path/to/script.sh",
        "timeout": 30
      }
    ],
    "PostToolUse": [
      {
        "type": "http",
        "url": "https://webhook.example.com/poyo",
        "method": "POST",
        "headers": {
          "Authorization": "Bearer token"
        }
      }
    ]
  }
}
```

---

## API 参考

### Poyo API 命名空间

所有插件 API 都在 `poyo` 命名空间下：

```lua
-- 核心功能
poyo.config.get(key)           -- 获取配置
poyo.config.set(key, value)    -- 设置配置
poyo.log.info(msg)             -- 日志输出
poyo.log.debug(msg)            -- 调试日志
poyo.log.error(msg)            -- 错误日志

-- HTTP 请求
poyo.http.get(url, headers)    -- GET 请求
poyo.http.post(url, body)      -- POST 请求

-- 文件系统
poyo.fs.read(path)             -- 读取文件
poyo.fs.write(path, content)   -- 写入文件
poyo.fs.exists(path)           -- 检查存在

-- 梦之国 (Dream Land)
poyo.land                      -- 工作目录
poyo.env.get(key)              -- 环境变量
poyo.env.set(key, value)       -- 设置环境变量
poyo.session.id()              -- 会话 ID

-- 能力系统
poyo.ability.use(name, input)  -- 使用能力
poyo.ability.list()            -- 列出能力
poyo.copy(enemy)               -- 复制能力！
poyo.inhale(target)            -- 吸入

-- 有趣的 API
poyo.say("message")            -- Poyo 说话
poyo.dance()                   -- Poyo 跳舞
poyo.poyo()                    -- Poyo 叫
```

### 工具 API

```go
// 工具接口
type Tool interface {
    Name() string
    Description() string
    InputSchema() ToolInputJSONSchema
    Call(ctx context.Context, input map[string]interface{},
         toolCtx *ToolUseContext, canUseTool CanUseToolFunc,
         progress ToolCallProgress) (*ToolResult, error)
}

// 注册工具
tools.DefaultRegistry.Register(&MyTool{})
```

---

## 插件系统

### 插件类型

Poyo 支持 5 种插件格式：

| 格式 | 运行时 | 说明 |
|------|--------|------|
| Lua Plugin | gopher-lua | 原生 Lua 插件 |
| WASM Plugin | wazero | WebAssembly 插件 |
| Script Plugin | Shell/Python/Node | 脚本插件 |
| MCP Plugin | JSON-RPC | Model Context Protocol |
| Compat Plugin | - | CC/OpenClaw 兼容 |

### Lua 插件示例

```lua
-- plugins/my-plugin/main.lua
function poyo.init()
    poyo.log.info("My plugin loaded!")
end

function poyo.abilities.custom_action(input)
    return {
        success = true,
        message = "Custom action executed: " .. input.text
    }
end
```

```json
// plugins/my-plugin/plugin.json
{
    "name": "my-plugin",
    "version": "1.0.0",
    "type": "lua",
    "description": "My custom plugin",
    "main": "main.lua"
}
```

### 插件发现

Poyo 会自动扫描以下目录：

```
~/.poyo/plugins/           # 用户插件
./plugins/                  # 项目插件
/path/in/POYO_PLUGIN_PATH/ # 环境变量指定
```

---

## 多 Agent 协作

### 场景式多 Agent

Poyo 支持场景式的多 Agent 协作模式：

```python
# 使用提示词自动生成场景团队
python test_multi_agent_scenario.py "开发一个在线商城系统"
```

### 预定义场景

| 场景 | Agent 配置 | 适用任务 |
|------|------------|----------|
| software_dev | Coordinator → Developer → Reviewer → Tester | 软件开发 |
| research | Coordinator → Researcher → Analyzer → Writer | 研究报告 |
| devops | Monitor → DevOps → Reviewer → Validator | 运维故障 |
| content_creation | Coordinator → Researcher → Writer → Reviewer | 内容创作 |
| data_analysis | Coordinator → Researcher → Analyzer → Validator | 数据分析 |
| bug_fix | Coordinator → Developer → Reviewer → Tester | Bug 修复 |

### Agent 角色

```python
class AgentRole(Enum):
    COORDINATOR = "coordinator"  # 协调者 - 任务分解和分配
    WORKER = "worker"            # 工作者 - 执行开发任务
    REVIEWER = "reviewer"        # 审核者 - 代码审核
    VALIDATOR = "validator"      # 验证者 - 测试验证
    RESEARCHER = "researcher"    # 研究者 - 信息收集
    DEVOPS = "devops"            # 运维者 - 故障修复
    MONITOR = "monitor"          # 监控者 - 问题发现
```

### 消息协作

```python
# Agent 间消息传递
team.send_message(
    from_agent="coordinator",
    to_agent="developer",
    type="task_assignment",
    content="实现用户登录功能"
)

# 广播消息
team.broadcast(
    from_agent="monitor",
    msg_type="alert",
    content="检测到服务异常"
)
```

---

## 测试指南

### 测试套件

| 测试模块 | 文件 | 测试数 | 覆盖范围 |
|----------|------|--------|----------|
| 上下文压缩 | test_context_compression.py | 28 | Token 阈值、压缩策略、信息保留 |
| 长期记忆 | test_long_term_memory.py | 37 | 存储、检索、命名空间、持久化 |
| 端到端记忆 | test_memory_e2e.py | 7 | 会话生命周期、并发访问 |
| 交互式对话 | test_interactive_memory.py | 8 | 多轮对话、记忆提取 |
| 多 Agent 场景 | test_multi_agent_scenario.py | 5 | 场景生成、Agent 协作 |
| **总计** | - | **85** | - |

### 运行测试

```bash
# 运行所有测试
python3 test_context_compression.py
python3 test_long_term_memory.py
python3 test_memory_e2e.py
python3 test_interactive_memory.py
python3 test_multi_agent_scenario.py

# 运行自定义场景
python3 test_multi_agent_scenario.py "你的任务描述"

# Go 单元测试
go test ./...
```

### 测试输出示例

```
======================================================================
🧪 Poyo 完整测试套件
======================================================================

▶ 测试1: 上下文压缩
  ✅ 通过: 28
  ❌ 失败: 0
  📈 通过率: 100.0%

▶ 测试2: 长期记忆
  ✅ 通过: 37
  ❌ 失败: 0
  📈 通过率: 100.0%

...

======================================================================
🎉 所有 85 个测试通过！
======================================================================
```

---

## 开发指南

### 开发环境

```bash
# 克隆项目
git clone https://github.com/kirbchino/poyo.git
cd poyo

# 安装开发依赖
go mod download

# 开发模式构建
make dev

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint
```

### 添加新工具

1. 在 `internal/tools/` 创建新文件：

```go
// internal/tools/my_tool.go
package tools

type MyTool struct {
    BaseTool
}

func NewMyTool() *MyTool {
    return &MyTool{
        BaseTool: BaseTool{
            name:        "MyTool",
            description: "My custom tool",
        },
    }
}

func (t *MyTool) Call(ctx context.Context, input map[string]interface{},
    toolCtx *ToolUseContext, canUseTool CanUseToolFunc,
    progress ToolCallProgress) (*ToolResult, error) {
    // 实现工具逻辑
    return &ToolResult{Data: result}, nil
}

func (t *MyTool) InputSchema() ToolInputJSONSchema {
    return ToolInputJSONSchema{
        Type: "object",
        Properties: map[string]map[string]interface{}{
            "input": {"type": "string"},
        },
    }
}
```

2. 注册工具：

```go
// internal/tools/registry.go
func InitializeBuiltinTools() {
    DefaultRegistry.Register(NewMyTool())
}
```

### 添加新命令

```go
// internal/commands/builtin.go
func init() {
    RegisterCommand(Command{
        Name:        "/mycommand",
        Description: "My custom command",
        Handler:     handleMyCommand,
        Category:    "custom",
    })
}

func handleMyCommand(ctx context.Context, args []string) error {
    // 命令逻辑
    return nil
}
```

### 添加新 Hook

```go
// 执行 Hook
result, err := hookExecutor.ExecuteHook(ctx, hooks.HookContext{
    Event:     hooks.PreToolUse,
    ToolName:  "Bash",
    ToolInput: input,
})

if result.StopReason != "" {
    // Hook 请求停止
}
```

---

## 附录

### 与 Claude Code 对比

| 功能 | Claude Code | Poyo | 说明 |
|------|-------------|------|------|
| 文件操作 | ✅ | ✅ | 完整 |
| Shell 执行 | ✅ | ✅ | 完整 |
| 子代理 | ✅ | ✅ | 4 种类型 |
| Hook 系统 | ✅ | ✅ | 26 种事件 |
| MCP 集成 | ✅ | ✅ | 5 种协议 |
| 权限系统 | ✅ | ✅ | 5 种模式 |
| Lua 插件 | ❌ | ✅ | Poyo 独有 |
| Vim 模式 | ✅ | ✅ | 完整 |
| 会话压缩 | ✅ | ✅ | 4 种策略 |
| 多 Agent | ❌ | ✅ | 场景式协作 |

### 常见问题

**Q: 如何切换模型？**

```bash
poyo -m claude-opus-4-6 "你的问题"
# 或在配置文件中设置
```

**Q: 如何调试？**

```bash
poyo --debug "你的问题"
```

**Q: 如何跳过权限确认？**

```bash
poyo -p accept-all "你的问题"
```

**Q: 如何持久化记忆？**

记忆会自动保存在 `.poyo/memory.json` 文件中。

### 许可证

MIT License

---

💚 **Poyo~! Portal for All Agents!** ☆彡
