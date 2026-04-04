# 💚 Poyo 插件系统 - 完整命名空间文档

## 🌟 生态系统概览

```
┌───────────────────────────────────────────────────────────────┐
│                     POYO 统一命名空间                           │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│   💚 核心功能                                                  │
│   ├── poyo.plugin.*     插件信息                              │
│   ├── poyo.config.*     配置                                  │
│   ├── poyo.log/debug/info/warn/error  日志系统               │
│   ├── poyo.json.*       JSON 处理                             │
│   ├── poyo.http.*       HTTP 请求                             │
│   ├── poyo.fs.*         文件系统                              │
│   ├── poyo.prompt.*     用户交互                              │
│   ├── poyo.cache.*      缓存                                  │
│   ├── poyo.hook.*       钩子                                  │
│   └── poyo.command.*    命令                                  │
│                                                               │
│   🌙 梦之国 Dream Land（卡比的家乡）                            │
│   ├── poyo.land          梦之国路径（工作目录）                │
│   ├── poyo.env.*         梦之国环境变量                       │
│   ├── poyo.context.*     梦之国上下文                         │
│   └── poyo.session.*     会话信息                             │
│                                                               │
│   ⭐ 能力系统 Copy Ability                                     │
│   ├── poyo.ability.*     能力系统核心                         │
│   ├── poyo.use()         使用能力的快捷方式                   │
│   ├── poyo.copy()        复制能力的快捷方式                   │
│   └── poyo.inhale()      吸入                                │
│                                                               │
│   🎮 有趣的 API                                                │
│   ├── poyo.say()         Poyo 说话！                          │
│   ├── poyo.dance()       Poyo 跳舞！                          │
│   └── poyo.poyo()        Poyo 叫！                            │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

## 💚 核心功能

### poyo.plugin - 插件信息

| API | 说明 |
|-----|------|
| `poyo.plugin.id` | 插件 ID |
| `poyo.plugin.name` | 插件名称 |
| `poyo.plugin.version` | 插件版本 |
| `poyo.plugin.path` | 插件路径 |

### poyo.config - 配置

| API | 说明 |
|-----|------|
| `poyo.config.*` | 访问插件配置 |

### poyo.log - 日志系统

| API | 说明 |
|-----|------|
| `poyo.log(level, message)` | 通用日志输出 |
| `poyo.debug(message)` | DEBUG 级别 |
| `poyo.info(message)` | INFO 级别 |
| `poyo.warn(message)` | WARN 级别 |
| `poyo.error(message)` | ERROR 级别 |

### poyo.json - JSON 处理

| API | 说明 |
|-----|------|
| `poyo.json.encode(value)` | 编码为 JSON |
| `poyo.json.decode(string)` | 解码 JSON |
| `poyo.json.pretty(value)` | 格式化 JSON |

### poyo.http - HTTP 请求

| API | 说明 |
|-----|------|
| `poyo.http.get(url, headers?)` | GET 请求 |
| `poyo.http.post(url, body, headers?)` | POST 请求 |

### poyo.fs - 文件系统

| API | 说明 |
|-----|------|
| `poyo.fs.read(path)` | 读取文件 |
| `poyo.fs.write(path, content)` | 写入文件 |
| `poyo.fs.exists(path)` | 检查存在 |
| `poyo.fs.list(path)` | 列出目录 |
| `poyo.fs.mkdir(path)` | 创建目录 |
| `poyo.fs.remove(path)` | 删除文件/目录 |

### poyo.prompt - 用户交互

| API | 说明 |
|-----|------|
| `poyo.prompt.select(message, options)` | 选择框 |
| `poyo.prompt.input(message, default?)` | 输入框 |
| `poyo.prompt.confirm(message, default?)` | 确认框 |

### poyo.cache - 缓存

| API | 说明 |
|-----|------|
| `poyo.cache.get(key)` | 获取缓存 |
| `poyo.cache.set(key, value, ttl?)` | 设置缓存 |
| `poyo.cache.delete(key)` | 删除缓存 |
| `poyo.cache.clear()` | 清空缓存 |

### poyo.hook - 钩子

| API | 说明 |
|-----|------|
| `poyo.hook.register(type, handler)` | 注册钩子 |
| `poyo.hook.types()` | 获取钩子类型列表 |

### poyo.command - 命令

| API | 说明 |
|-----|------|
| `poyo.command.register(name, handler)` | 注册命令 |
| `poyo.command.list()` | 获取命令列表 |

## 🌙 梦之国 Dream Land

### poyo.land - 梦之国路径

| API | 说明 |
|-----|------|
| `poyo.land` | 梦之国路径（工作目录）|

### poyo.env - 环境变量

| API | 说明 |
|-----|------|
| `poyo.env.get(key)` | 获取环境变量 |
| `poyo.env.set(key, value)` | 设置环境变量 |
| `poyo.env.list()` | 列出所有环境变量 |

### poyo.context - 上下文

| API | 说明 |
|-----|------|
| `poyo.context.get()` | 获取完整上下文 |
| `poyo.context.set(key, value)` | 设置上下文值 |

### poyo.session - 会话信息

| API | 说明 |
|-----|------|
| `poyo.session.id()` | 获取会话 ID |
| `poyo.session.user_id()` | 获取用户 ID |

## ⭐ 能力系统 Copy Ability

### poyo.ability - 能力核心

| API | 说明 |
|-----|------|
| `poyo.ability.use(name, input)` | 使用能力 |
| `poyo.ability.list()` | 列出所有能力 |
| `poyo.ability.copy(enemy)` | 复制能力！|

### 能力快捷方式

| API | 说明 |
|-----|------|
| `poyo.use(name, input)` | 使用能力的快捷方式 |
| `poyo.copy(enemy)` | 复制能力的快捷方式 |
| `poyo.inhale(path)` | 吸入（分析代码库）|

## 🎮 有趣的 API

| API | 说明 |
|-----|------|
| `poyo.say(message)` | Poyo 说话！ |
| `poyo.dance()` | Poyo 跳舞！ |
| `poyo.poyo()` | Poyo 叫！ |

## 📋 完整示例

```lua
-- 💚 Poyo 插件示例
local M = {}

function M.init()
    -- 💚 核心 API
    poyo.log("info", "插件初始化！")
    poyo.say("欢迎来到梦之国！")

    -- 🌙 梦之国
    local land = poyo.land
    local sid = poyo.session.id()
    poyo.log("info", "Dream Land: " .. land)

    -- ⭐ 能力系统
    local abilities = poyo.ability.list()

    -- 使用能力
    local result = poyo.use("Read", {path = "main.go"})

    -- 复制能力
    local success, msg = poyo.copy("fire-enemy")

    poyo.poyo()
end

-- 钩子示例
function M.on_pre_tool_use(input)
    poyo.log("debug", "工具执行: " .. input.tool)
    return {blocked = false}
end

-- 吸入示例
function M.inhale(path)
    poyo.inhale(path)
    poyo.say("吸入 " .. path)

    local entries = poyo.fs.list(path)
    poyo.dance()

    return {entries = entries}
end

M.init()
return M
```

---

💚 **Poyo~! Welcome to Dream Land!** ☆彡
