-- 🧪 Host API Test Plugin
-- 测试插件反向调用 Host 的所有能力

local M = {}

-- ═══════════════════════════════════════════════════════
-- 🔧 初始化
-- ═══════════════════════════════════════════════════════

function M.init()
    poyo.log("info", "🧪 Host API Test Plugin initialized")
    poyo.log("info", string.format("Plugin: %s v%s", poyo.plugin.name, poyo.plugin.version))
    poyo.say("准备测试所有 Host API!")
end

-- ═══════════════════════════════════════════════════════
-- 🎯 测试：工具调用 (poyo.use / poyo.ability.use)
-- ═══════════════════════════════════════════════════════

function M.test_tool_call(args)
    poyo.say("测试工具调用能力!")
    poyo.dance()

    local results = {}

    -- 测试 1: 调用 Bash 工具
    poyo.log("info", "测试 1: 调用 Bash 工具")
    local bash_result, err = poyo.use("Bash", {
        command = "echo 'Hello from plugin!'",
        description = "Test bash call from plugin"
    })
    results.bash = {
        success = err == nil,
        result = bash_result,
        error = err
    }

    -- 测试 2: 调用 Read 工具
    poyo.log("info", "测试 2: 调用 Read 工具")
    local read_result, read_err = poyo.use("Read", {
        path = "plugin.json"
    })
    results.read = {
        success = read_err == nil,
        has_content = read_result ~= nil and read_result.content ~= nil,
        error = read_err
    }

    -- 测试 3: 调用 Glob 工具
    poyo.log("info", "测试 3: 调用 Glob 工具")
    local glob_result, glob_err = poyo.use("Glob", {
        pattern = "*.lua"
    })
    results.glob = {
        success = glob_err == nil,
        error = glob_err
    }

    -- 测试 4: 调用 Grep 工具
    poyo.log("info", "测试 4: 调用 Grep 工具")
    local grep_result, grep_err = poyo.use("Grep", {
        pattern = "poyo",
        path = "."
    })
    results.grep = {
        success = grep_err == nil,
        error = grep_err
    }

    -- 测试 5: 列出可用能力
    local abilities = poyo.ability.list()
    results.available_abilities = abilities

    poyo.say("工具调用测试完成!")

    return {
        test = "tool_call",
        results = results,
        passed = results.bash.success and results.read.success
    }
end

-- ═══════════════════════════════════════════════════════
-- 📁 测试：文件操作 (poyo.fs)
-- ═══════════════════════════════════════════════════════

function M.test_file_ops(args)
    poyo.say("测试文件操作能力!")

    local results = {}
    local test_file = "test_output.txt"
    local test_content = "Hello from Poyo Plugin!\n测试中文内容。\n"

    -- 测试 1: 写文件
    poyo.log("info", "测试 1: 写文件")
    local write_ok, write_err = poyo.fs.write(test_file, test_content)
    results.write = {
        success = write_ok == true,
        error = write_err
    }

    -- 测试 2: 检查文件存在
    poyo.log("info", "测试 2: 检查文件存在")
    local exists = poyo.fs.exists(test_file)
    results.exists = {
        success = exists == true
    }

    -- 测试 3: 读取文件
    poyo.log("info", "测试 3: 读取文件")
    local content, read_err = poyo.fs.read(test_file)
    results.read = {
        success = read_err == nil and content ~= nil,
        content_match = content == test_content,
        error = read_err
    }

    -- 测试 4: 列出目录
    poyo.log("info", "测试 4: 列出目录")
    local entries, list_err = poyo.fs.list(".")
    results.list = {
        success = list_err == nil,
        count = entries and #entries or 0,
        error = list_err
    }

    -- 测试 5: 创建目录
    poyo.log("info", "测试 5: 创建目录")
    local mkdir_ok, mkdir_err = poyo.fs.mkdir("test_dir/subdir")
    results.mkdir = {
        success = mkdir_ok == true,
        error = mkdir_err
    }

    -- 测试 6: 删除文件
    poyo.log("info", "测试 6: 删除文件")
    local rm_ok, rm_err = poyo.fs.remove(test_file)
    results.remove = {
        success = rm_ok == true,
        error = rm_err
    }

    -- 清理
    poyo.fs.remove("test_dir")

    poyo.say("文件操作测试完成!")

    return {
        test = "file_ops",
        results = results,
        passed = results.write.success and results.read.success and results.exists.success
    }
end

-- ═══════════════════════════════════════════════════════
-- 🌐 测试：HTTP 请求 (poyo.http)
-- ═══════════════════════════════════════════════════════

function M.test_http(args)
    poyo.say("测试 HTTP 请求能力!")

    local results = {}

    -- 测试 1: GET 请求
    poyo.log("info", "测试 1: GET 请求")
    local get_result, get_err = poyo.http.get("https://httpbin.org/get", {
        ["X-Test-Header"] = "poyo-test"
    })
    results.get = {
        success = get_err == nil,
        status_code = get_result and get_result.statusCode or 0,
        error = get_err
    }

    -- 测试 2: POST 请求
    poyo.log("info", "测试 2: POST 请求")
    local post_result, post_err = poyo.http.post("https://httpbin.org/post", {
        message = "Hello from Poyo!",
        test = true
    }, {
        ["X-Custom-Header"] = "poyo-custom"
    })
    results.post = {
        success = post_err == nil,
        status_code = post_result and post_result.statusCode or 0,
        error = post_err
    }

    poyo.say("HTTP 请求测试完成!")

    return {
        test = "http",
        results = results,
        passed = results.get.success and results.post.success
    }
end

-- ═══════════════════════════════════════════════════════
-- 💾 测试：缓存 (poyo.cache)
-- ═══════════════════════════════════════════════════════

function M.test_cache(args)
    poyo.say("测试缓存能力!")

    local results = {}
    local test_key = "test_key_" .. os.time()
    local test_value = {name = "poyo", count = 42}

    -- 测试 1: 设置缓存
    poyo.log("info", "测试 1: 设置缓存")
    poyo.cache.set(test_key, test_value, 60)
    results.set = {success = true}

    -- 测试 2: 获取缓存
    poyo.log("info", "测试 2: 获取缓存")
    local cached, found = poyo.cache.get(test_key)
    results.get = {
        success = found == true,
        value_match = cached and cached.name == "poyo"
    }

    -- 测试 3: 删除缓存
    poyo.log("info", "测试 3: 删除缓存")
    poyo.cache.delete(test_key)
    local _, still_found = poyo.cache.get(test_key)
    results.delete = {
        success = still_found == false
    }

    poyo.say("缓存测试完成!")

    return {
        test = "cache",
        results = results,
        passed = results.get.success and results.delete.success
    }
end

-- ═══════════════════════════════════════════════════════
-- 📋 测试：JSON (poyo.json)
-- ═══════════════════════════════════════════════════════

function M.test_json(args)
    poyo.say("测试 JSON 能力!")

    local results = {}
    local test_data = {
        name = "poyo",
        version = "1.0",
        features = {"tools", "hooks", "commands"},
        nested = {
            key = "value"
        }
    }

    -- 测试 1: 编码
    poyo.log("info", "测试 1: JSON 编码")
    local encoded, enc_err = poyo.json.encode(test_data)
    results.encode = {
        success = enc_err == nil,
        is_string = type(encoded) == "string"
    }

    -- 测试 2: 解码
    poyo.log("info", "测试 2: JSON 解码")
    local decoded, dec_err = poyo.json.decode(encoded)
    results.decode = {
        success = dec_err == nil,
        name_match = decoded and decoded.name == "poyo"
    }

    -- 测试 3: 美化输出
    poyo.log("info", "测试 3: JSON 美化")
    local pretty, pretty_err = poyo.json.pretty(test_data)
    results.pretty = {
        success = pretty_err == nil,
        is_formatted = pretty and string.find(pretty, "\n") ~= nil
    }

    poyo.say("JSON 测试完成!")

    return {
        test = "json",
        results = results,
        passed = results.encode.success and results.decode.success
    }
end

-- ═══════════════════════════════════════════════════════
-- 🔧 测试：环境变量 (poyo.env)
-- ═══════════════════════════════════════════════════════

function M.test_env(args)
    poyo.say("测试环境变量能力!")

    local results = {}

    -- 测试 1: 设置环境变量
    poyo.log("info", "测试 1: 设置环境变量")
    poyo.env.set("POYO_TEST_VAR", "test_value")
    results.set = {success = true}

    -- 测试 2: 获取环境变量
    poyo.log("info", "测试 2: 获取环境变量")
    local value = poyo.env.get("POYO_TEST_VAR")
    results.get = {
        success = value == "test_value",
        value = value
    }

    -- 测试 3: 列出所有环境变量
    poyo.log("info", "测试 3: 列出环境变量")
    local all_env = poyo.env.list()
    results.list = {
        success = all_env ~= nil,
        has_path = all_env and all_env["PATH"] ~= nil
    }

    poyo.say("环境变量测试完成!")

    return {
        test = "env",
        results = results,
        passed = results.get.success
    }
end

-- ═══════════════════════════════════════════════════════
-- 🧪 测试所有 API
-- ═══════════════════════════════════════════════════════

function M.test_all_apis(args)
    poyo.poyo()
    poyo.say("开始全面测试所有 Host API!")

    local all_results = {
        plugin_info = {
            id = poyo.plugin.id,
            name = poyo.plugin.name,
            version = poyo.plugin.version,
            path = poyo.plugin.path
        },
        dream_land = poyo.land,
        config = poyo.config
    }

    -- 运行所有测试
    all_results.tool_call = M.test_tool_call(args)
    all_results.file_ops = M.test_file_ops(args)
    all_results.http = M.test_http(args)
    all_results.cache = M.test_cache(args)
    all_results.json = M.test_json(args)
    all_results.env = M.test_env(args)

    -- 计算通过率
    local passed = 0
    local total = 0
    for name, result in pairs(all_results) do
        if type(result) == "table" and result.passed ~= nil then
            total = total + 1
            if result.passed then
                passed = passed + 1
            end
        end
    end

    all_results.summary = {
        total_tests = total,
        passed = passed,
        failed = total - passed,
        pass_rate = total > 0 and math.floor(passed / total * 100) or 0
    }

    poyo.say(string.format("测试完成! %d/%d 通过 (%d%%)", passed, total, all_results.summary.pass_rate))
    poyo.dance()

    return all_results
end

-- ═══════════════════════════════════════════════════════
-- 🪝 Hooks
-- ═══════════════════════════════════════════════════════

function M.PreToolUse(input)
    local tool = input.tool or input.args and input.args.tool or "unknown"
    poyo.log("debug", "[Hook] Tool about to execute: " .. tool)

    -- 测试：在钩子中访问 host API
    local env_test = poyo.env.get("POYO_API_VERSION")
    poyo.log("debug", "[Hook] API Version: " .. (env_test or "not set"))

    return {
        blocked = false,
        message = "Host API test plugin monitoring",
        source = "host-api-test"
    }
end

function M.PostToolUse(input)
    local tool = input.tool or "unknown"
    local success = input.success or false

    poyo.log("debug", string.format("[Hook] Tool %s finished: %s", tool, success and "success" or "failed"))

    -- 测试：在钩子中使用缓存
    local key = "tool_" .. tool .. "_count"
    local count, _ = poyo.cache.get(key)
    count = (count or 0) + 1
    poyo.cache.set(key, count)

    return {
        logged = true,
        tool_count = count
    }
end

-- ═══════════════════════════════════════════════════════
-- 初始化
-- ═══════════════════════════════════════════════════════

M.init()

return M
