-- 💚 Poyo Example Plugin
-- 展示统一的 poyo 命名空间 API

local M = {}

-- 插件初始化
function M.init()
    poyo.log("info", "💚 Poyo Plugin initialized!")
    poyo.log("info", string.format("Plugin: %s v%s", poyo.plugin.name, poyo.plugin.version))
    poyo.poyo()
end

-- ═══════════════════════════════════════════════════════
-- 💚 POYO 核心 API 示例
-- ═══════════════════════════════════════════════════════

function M.demo_core()
    -- 日志
    poyo.log("info", "这是一条日志")
    poyo.debug("调试信息")
    poyo.warn("警告信息")
    poyo.error("错误信息")

    -- JSON
    local data = {name = "poyo", version = "1.0"}
    local json_str = poyo.json.encode(data)
    poyo.log("info", "JSON: " .. json_str)

    -- 文件系统
    local exists = poyo.fs.exists("README.md")
    if exists then
        local content = poyo.fs.read("README.md")
        poyo.log("info", "读取到 " .. #content .. " 字节")
    end

    -- 有趣的 API
    poyo.say("Hello from Poyo!")
    poyo.dance()
    poyo.poyo()
end

-- ═══════════════════════════════════════════════════════
-- 🌙 梦之国 API 示例 (poyo.land / poyo.env / poyo.session)
-- ═══════════════════════════════════════════════════════

function M.demo_dream()
    -- 梦之国路径（工作目录）
    local land = poyo.land
    poyo.log("info", "Dream Land: " .. land)

    -- 环境变量
    local path = poyo.env.get("PATH")
    poyo.log("info", "PATH: " .. (path or "not set"))

    -- 设置环境变量
    poyo.env.set("POYO_MODE", "happy")

    -- 列出所有环境变量
    local all_env = poyo.env.list()

    -- 上下文
    local ctx = poyo.context.get()
    poyo.context.set("custom_key", "custom_value")

    -- 会话信息
    local session_id = poyo.session.id()
    local user_id = poyo.session.user_id()
    poyo.log("info", "Session: " .. session_id)
    poyo.log("info", "User: " .. user_id)
end

-- ═══════════════════════════════════════════════════════
-- ⭐ 能力系统 API 示例 (poyo.ability / poyo.use / poyo.copy)
-- ═══════════════════════════════════════════════════════

function M.demo_ability()
    -- 列出所有能力
    local abilities = poyo.ability.list()
    poyo.log("info", "可用能力: " .. #abilities .. " 个")

    -- 使用能力
    local result = poyo.ability.use("Read", {path = "README.md"})

    -- 快捷方式：直接用 poyo.use
    local result2 = poyo.use("Bash", {command = "ls -la"})

    -- 复制能力！（卡比的标志性能力）
    local success, msg = poyo.copy("fire-enemy")
    if success then
        poyo.say("我复制了能力! " .. msg)
        poyo.dance()
    end
end

-- ═══════════════════════════════════════════════════════
-- 🔄 钩子与命令示例
-- ═══════════════════════════════════════════════════════

-- 注册钩子
function M.setup_hooks()
    poyo.hook.register("PreToolUse", function(input)
        poyo.log("debug", "工具即将执行: " .. (input.tool or "unknown"))
        return {blocked = false}
    end)

    poyo.hook.register("PostToolUse", function(input)
        if input.success then
            poyo.log("debug", "工具执行成功")
        end
        return {}
    end)
end

-- 注册命令
function M.setup_commands()
    poyo.command.register("poyo-hello", function(input)
        poyo.say("你好！我是 Poyo~!")
        return {message = "Hello from Poyo!"}
    end)
end

-- ═══════════════════════════════════════════════════════
-- 综合示例：吸入并分析
-- ═══════════════════════════════════════════════════════

function M.inhale(input)
    local path = input.path or "."
    poyo.inhale(path)
    poyo.say("吸入 " .. path .. " 中...")

    local exists = poyo.fs.exists(path)
    if not exists then
        return {success = false, error = "路径不存在"}
    end

    local entries = poyo.fs.list(path)
    poyo.dance()

    return {
        success = true,
        result = {
            path = path,
            entries = entries,
            dream_land = poyo.land,
            session = poyo.session.id()
        }
    }
end

-- 钩子：工具使用前
function M.on_pre_tool_use(input)
    local tool = input.tool
    poyo.log("debug", "Poyo 监控: " .. tool)
    return {blocked = false, modified = false}
end

-- 初始化
M.init()
M.setup_hooks()
M.setup_commands()

return M
