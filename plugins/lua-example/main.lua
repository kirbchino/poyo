-- 💚 Poyo Lua Plugin Example
-- 展示如何编写 Poyo Lua 插件

-- 获取插件配置
local config = poyo.config
local plugin = poyo.plugin

-- 日志输出
poyo.log("info", (config.prefix or "") .. " 加载插件: " .. plugin.name)
poyo.log("debug", "插件路径: " .. plugin.path)

-- ═══════════════════════════════════════════════════════
-- 钩子示例
-- ═══════════════════════════════════════════════════════

-- 钩子: PreToolUse
function PreToolUse(input)
    poyo.log("info", "PreToolUse 钩子触发")
    poyo.log("debug", "工具: " .. (input.tool or "unknown"))

    -- 示例: 阻止某些工具
    -- if input.tool == "Bash" then
    --     return {
    --         blocked = true,
    --         reason = "此上下文中禁止使用 Bash"
    --     }
    -- end

    return {
        blocked = false,
        message = "允许执行工具"
    }
end

-- 钩子: PostToolUse
function PostToolUse(input)
    poyo.log("info", "PostToolUse 钩子触发")

    if input.success then
        poyo.log("info", "工具执行成功")
    else
        poyo.log("warn", "工具执行失败: " .. (input.error or "未知错误"))
    end

    return {logged = true}
end

-- ═══════════════════════════════════════════════════════
-- 工具示例
-- ═══════════════════════════════════════════════════════

-- 工具: lua_greet
function lua_greet(input)
    local name = input.name or "World"
    local greeting = "你好, " .. name .. "!"

    poyo.log("info", greeting)
    poyo.say(greeting)

    return {
        greeting = greeting,
        name = name,
        plugin = plugin.name,
        version = plugin.version
    }
end

-- 工具: process_text
function process_text(input)
    local text = input.text or ""
    local operation = input.operation or "upper"

    local result
    if operation == "upper" then
        result = string.upper(text)
    elseif operation == "lower" then
        result = string.lower(text)
    elseif operation == "reverse" then
        result = string.reverse(text)
    elseif operation == "len" then
        result = tostring(string.len(text))
    else
        result = text
    end

    return {
        original = text,
        operation = operation,
        result = result
    }
end

-- 工具: calculate
function calculate(input)
    local a = input.a or 0
    local b = input.b or 0
    local op = input.op or "add"

    local result
    if op == "add" then
        result = a + b
    elseif op == "sub" then
        result = a - b
    elseif op == "mul" then
        result = a * b
    elseif op == "div" then
        if b == 0 then
            return {error = "除数不能为零"}
        end
        result = a / b
    else
        return {error = "未知操作: " .. op}
    end

    return {
        a = a,
        b = b,
        operation = op,
        result = result
    }
end

-- ═══════════════════════════════════════════════════════
-- 命令示例
-- ═══════════════════════════════════════════════════════

-- 命令: /lua-hello
function lua_hello(input)
    poyo.poyo()
    return {
        message = "Hello from Poyo Lua plugin!",
        available_commands = {
            "/lua-greet <name> - 问候某人",
            "/lua-info - 显示插件信息"
        }
    }
end

-- ═══════════════════════════════════════════════════════
-- 使用能力系统示例
-- ═══════════════════════════════════════════════════════

function demo_ability()
    -- 使用能力
    local result = poyo.use("Read", {path = "README.md"})

    -- 复制能力
    local success, msg = poyo.copy("example-ability")

    -- 列出所有能力
    local abilities = poyo.ability.list()

    return {
        success = success,
        message = msg,
        ability_count = #abilities
    }
end

-- ═══════════════════════════════════════════════════════
-- 梦之国 API 示例
-- ═══════════════════════════════════════════════════════

function demo_dream()
    -- 获取梦之国路径
    local land = poyo.land

    -- 获取会话信息
    local sid = poyo.session.id()
    local uid = poyo.session.user_id()

    -- 环境变量
    local path_env = poyo.env.get("PATH")
    poyo.env.set("POYO_DEMO", "true")

    -- 上下文
    poyo.context.set("demo_key", "demo_value")
    local ctx = poyo.context.get()

    return {
        land = land,
        session_id = sid,
        user_id = uid
    }
end

-- 初始化
poyo.log("info", "插件初始化完成")
poyo.say("Poyo 插件已加载~!")

-- 返回插件元数据
return {
    name = plugin.name,
    version = plugin.version,
    author = plugin.author,
    description = plugin.description,
    hooks = {"PreToolUse", "PostToolUse"},
    tools = {"lua_greet", "process_text", "calculate"},
    commands = {"lua_hello"}
}
