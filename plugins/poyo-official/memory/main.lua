-- 🧠 Poyo Memory Plugin
-- 持久化记忆管理插件

local M = {}
local memory_store = {}
local config = {
    max_memory_size = 10000,
    default_namespace = "default",
    persistence_file = ".poyo/memory.json"
}

-- ═══════════════════════════════════════════════════════
-- 🔧 初始化
-- ═══════════════════════════════════════════════════════

function M.init(plugin_config)
    if plugin_config then
        for k, v in pairs(plugin_config) do
            config[k] = v
        end
    end

    -- 加载持久化存储
    M.load_from_file()
    poyo.log("info", "🧠 Memory Plugin initialized")
end

-- ═══════════════════════════════════════════════════════
-- 🎯 Tool Handlers
-- ═══════════════════════════════════════════════════════

-- 存储记忆
function M.store(args)
    local key = args.key
    local value = args.value
    local namespace = args.namespace or config.default_namespace

    if not key or not value then
        return {success = false, error = "key and value are required"}
    end

    -- 检查大小限制
    local total_size = 0
    for ns, _ in pairs(memory_store) do
        for k, v in pairs(memory_store[ns]) do
            total_size = total_size + #k + #v
        end
    end

    if total_size + #key + #value > config.max_memory_size then
        return {success = false, error = "memory size limit exceeded"}
    end

    -- 初始化命名空间
    if not memory_store[namespace] then
        memory_store[namespace] = {}
    end

    -- 存储记忆
    memory_store[namespace][key] = {
        value = value,
        timestamp = os.time(),
        access_count = 0
    }

    -- 持久化
    M.save_to_file()

    poyo.log("info", string.format("Stored memory: %s in namespace %s", key, namespace))

    return {
        success = true,
        key = key,
        namespace = namespace,
        size = #value
    }
end

-- 检索记忆
function M.retrieve(args)
    local key = args.key
    local namespace = args.namespace or config.default_namespace

    if not key then
        return {success = false, error = "key is required"}
    end

    if not memory_store[namespace] then
        return {success = false, error = "namespace not found: " .. namespace}
    end

    local entry = memory_store[namespace][key]
    if not entry then
        return {success = false, error = "memory not found: " .. key}
    end

    -- 更新访问计数
    entry.access_count = entry.access_count + 1

    poyo.log("info", string.format("Retrieved memory: %s", key))

    return {
        success = true,
        key = key,
        value = entry.value,
        namespace = namespace,
        timestamp = entry.timestamp,
        access_count = entry.access_count
    }
end

-- 列出记忆
function M.list(args)
    local namespace = args.namespace
    local result = {}

    if namespace then
        -- 列出特定命名空间
        if memory_store[namespace] then
            for key, entry in pairs(memory_store[namespace]) do
                table.insert(result, {
                    key = key,
                    namespace = namespace,
                    size = #entry.value,
                    timestamp = entry.timestamp,
                    access_count = entry.access_count
                })
            end
        end
    else
        -- 列出所有
        for ns, entries in pairs(memory_store) do
            for key, entry in pairs(entries) do
                table.insert(result, {
                    key = key,
                    namespace = ns,
                    size = #entry.value,
                    timestamp = entry.timestamp,
                    access_count = entry.access_count
                })
            end
        end
    end

    -- 排序：按时间戳降序
    table.sort(result, function(a, b)
        return a.timestamp > b.timestamp
    end)

    poyo.log("info", string.format("Listed %d memories", #result))

    return {
        success = true,
        memories = result,
        total = #result
    }
end

-- 删除记忆
function M.forget(args)
    local key = args.key
    local namespace = args.namespace or config.default_namespace

    if not key then
        return {success = false, error = "key is required"}
    end

    if not memory_store[namespace] then
        return {success = false, error = "namespace not found: " .. namespace}
    end

    if not memory_store[namespace][key] then
        return {success = false, error = "memory not found: " .. key}
    end

    memory_store[namespace][key] = nil

    -- 清理空命名空间
    local empty = true
    for _ in pairs(memory_store[namespace]) do
        empty = false
        break
    end
    if empty then
        memory_store[namespace] = nil
    end

    -- 持久化
    M.save_to_file()

    poyo.log("info", string.format("Forgot memory: %s", key))

    return {
        success = true,
        key = key,
        namespace = namespace
    }
end

-- ═══════════════════════════════════════════════════════
-- 🪝 Hooks
-- ═══════════════════════════════════════════════════════

function M.on_pre_tool_use(input)
    local tool = input.tool or "unknown"
    poyo.log("debug", string.format("Memory plugin tracking tool: %s", tool))
    return {blocked = false}
end

-- ═══════════════════════════════════════════════════════
-- 💾 Persistence
-- ═══════════════════════════════════════════════════════

function M.save_to_file()
    local json_str = poyo.json.encode(memory_store)
    poyo.fs.write(config.persistence_file, json_str)
end

function M.load_from_file()
    if poyo.fs.exists(config.persistence_file) then
        local content = poyo.fs.read(config.persistence_file)
        if content and #content > 0 then
            local decoded = poyo.json.decode(content)
            if decoded then
                memory_store = decoded
            end
        end
    end
end

-- ═══════════════════════════════════════════════════════
-- 🚀 Commands
-- ═══════════════════════════════════════════════════════

function M.memory_stats()
    local namespaces = 0
    local total_keys = 0
    local total_size = 0

    for ns, entries in pairs(memory_store) do
        namespaces = namespaces + 1
        for key, entry in pairs(entries) do
            total_keys = total_keys + 1
            total_size = total_size + #key + #entry.value
        end
    end

    return {
        success = true,
        stats = {
            namespaces = namespaces,
            total_keys = total_keys,
            total_size = total_size,
            max_size = config.max_memory_size,
            usage_percent = math.floor(total_size / config.max_memory_size * 100)
        }
    }
end

-- ═══════════════════════════════════════════════════════
-- 📋 Exports (CC 格式)
-- ═══════════════════════════════════════════════════════

M.store = M.store
M.retrieve = M.retrieve
M.list = M.list
M.forget = M.forget
M["memory_stats"] = M.memory_stats

return M
