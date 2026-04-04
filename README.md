# 💚 Poyo

> **Portal Of Your Orchestrated Omnibus-agents**
>
> 编排你所有 Agent 的统一门户平台

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 🌀 为什么叫 Poyo？

就像星之卡比能吸入并复制敌人的能力一样，**Poyo** 可以：

- 🌀 **Inhale 吸入** — 摄取代码库、文档、知识
- ⭐ **Copy 复制** — 学习模式并应用到新场景
- 💪 **Ability 能力** — 使用工具完成各种任务
- 🌙 **Dream Land 梦之国** — 在统一环境中自由探索
- 🔌 **Omnibus 包容** — 兼容 CC、OpenClaw、MCP 等多种插件格式

**Slogan**: 🌀 Poyo — Portal for All Agents / 吸入万物，释放可能

## ✨ 核心特性

### 🔧 工具系统
- 25+ 内置工具（Read, Write, Edit, Bash, Glob, Grep 等）
- AgentTool 子代理系统（4种 Agent 类型）
- WebFetch/WebSearch 网络能力

### 🪝 Hook 系统
- 26 种事件类型
- HTTP Hook 支持
- SSRF 防护
- 异步执行

### 🔐 权限系统
- 6 层规则来源
- 用户确认机制
- 沙箱隔离
- 命令白名单

### 🔌 MCP 集成
- 5 种传输协议
- OAuth 2.0 认证
- 资源、提示、工具支持

### 📝 命令系统
- 109 个斜杠命令
- 命令别名支持
- 分类管理

### 📦 其他特性
- Vim 编辑模式
- Jupyter Notebook 支持
- 会话压缩
- 计算机使用工具
- 多代理团队协作
- 文件历史/撤销

## 📊 项目架构

```
poyo/
├── cmd/                    # 入口命令
├── internal/
│   ├── agent/             # 子代理系统
│   ├── brief/             # 报告生成
│   ├── commands/          # 命令系统 (109命令)
│   ├── compactor/         # 会话压缩
│   ├── computeruse/       # 计算机使用
│   ├── filehistory/       # 文件历史
│   ├── hooks/             # Hook 系统
│   ├── interaction/       # 用户交互
│   ├── lsp/               # LSP 工具
│   ├── mcp/               # MCP 集成
│   ├── notebook/          # Notebook 编辑
│   ├── permission/        # 权限系统
│   ├── sandbox/           # 沙箱隔离
│   ├── session/           # 会话管理
│   ├── task/              # 任务管理
│   ├── team/              # 多代理团队
│   ├── tools/             # 核心工具
│   └── vim/               # Vim 模式
├── plugins/               # 插件示例
├── tests/                 # 端到端测试
└── docs/                  # 文档
```

## 🚀 快速开始

```bash
# 克隆仓库
git clone https://github.com/kirbchino/poyo.git
cd poyo

# 构建
go build -o poyo ./cmd/poyo

# 运行
./poyo "帮我实现一个 REST API"

# 交互模式
./poyo -i

# 查看帮助
./poyo /help
```

## 🔌 插件兼容性

Poyo 的 **Omnibus** 特性支持多种插件格式：

| 格式 | 状态 | 说明 |
|------|------|------|
| Lua Plugin | ✅ | 内置 gopher-lua VM，原生支持 |
| MCP (Model Context Protocol) | ✅ | JSON-RPC 2.0 协议 |
| Poyo Plugin | ✅ | 原生插件格式 |
| Claude Code Plugin | ✅ | 兼容 Claude Code 插件格式 |
| OpenClaw Plugin | ✅ | 兼容 OpenClaw 插件格式 |
| Script Plugin | ✅ | Shell/Python/Node.js 脚本 |

## 🌀 内置工具（Copy Abilities）

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
| WebSearch | 🔎 Search | 光束搜索 |
| Skill | ⭐ Copy | 技能复制 |
| NotebookEdit | 📓 Notebook | 笔记编辑 |

## 📝 命令系统

Poyo 提供 109 个斜杠命令，涵盖：

- **Session 管理**: /help, /clear, /compact, /history, /save, /load
- **配置**: /config, /model, /theme, /set, /get, /reset
- **Git & Code**: /commit, /review, /branch, /checkout, /merge, /rebase
- **文件操作**: /ls, /tree, /grep, /find, /read, /edit
- **构建测试**: /test, /build, /run, /format, /lint
- **模式切换**: /plan, /fast, /auto, /vim, /bughunter
- **集成**: /jira, /notion, /slack, /web, /terminal

## 🔒 安全特性

- 沙箱隔离执行
- 用户权限确认
- SSRF 防护
- 敏感信息过滤
- 命令白名单

## 📦 统一命名空间

所有 API 都在 `poyo` 命名空间下：

```lua
-- 核心功能
poyo.config.*           -- 配置
poyo.log(level, msg)    -- 日志
poyo.http.get/post      -- HTTP
poyo.fs.read/write      -- 文件系统

-- 梦之国 (Dream Land)
poyo.land               -- 工作目录
poyo.env.get/set        -- 环境变量
poyo.session.id()       -- 会话 ID

-- 能力系统
poyo.ability.use(name, input)  -- 使用能力
poyo.ability.list()            -- 列出能力
poyo.copy(enemy)               -- 复制能力！
```

## 📄 许可证

MIT License

---

💚 **Poyo~! Portal for All Agents!** ☆彡
