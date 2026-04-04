#!/bin/bash
# 🐚 Poyo Shell Script Plugin Example
# 展示 Shell 脚本插件的所有能力

set -e

# ═══════════════════════════════════════════════════════
# 🔧 Poyo API Helper Functions
# ═══════════════════════════════════════════════════════

# 从 stdin 读取 JSON 输入
poyo_get_input() {
    cat
}

# 输出 JSON 结果
poyo_output() {
    local success="$1"
    local result="$2"
    local error="$3"

    if [ -n "$error" ]; then
        echo "{\"success\": false, \"error\": \"$error\"}"
    else
        echo "{\"success\": $success, \"result\": $result}"
    fi
}

# 日志输出
poyo_log() {
    local level="$1"
    local message="$2"
    echo "[$level] $message" >&2
}

# Poyo 说话
poyo_say() {
    local message="$1"
    echo "💚 Poyo 说: $message" >&2
}

# ═══════════════════════════════════════════════════════
# 🎯 Tool Implementations
# ═══════════════════════════════════════════════════════

tool_shell_info() {
    local args="$1"

    poyo_say "获取系统信息"

    local os_name=$(uname -s)
    local os_version=$(uname -r)
    local hostname=$(hostname)
    local current_dir=$(pwd)
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # 构建结果 JSON
    local result=$(cat <<EOF
{
    "os": "$os_name",
    "version": "$os_version",
    "hostname": "$hostname",
    "current_dir": "$current_dir",
    "timestamp": "$timestamp",
    "dream_land": "$POYO_DREAM_LAND",
    "plugin_id": "$POYO_PLUGIN_ID"
}
EOF
)

    poyo_output "true" "$result"
}

tool_shell_greet() {
    local args="$1"
    local name=$(echo "$args" | grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*: *"\([^"]*\)".*/\1/')
    name=${name:-"World"}

    poyo_say "你好, $name!"

    poyo_output "true" "{\"greeting\": \"你好, $name! 来自 Shell 插件~\", \"plugin\": \"$POYO_PLUGIN_NAME\"}"
}

tool_shell_execute() {
    local args="$1"
    local cmd=$(echo "$args" | grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*: *"\([^"]*\)".*/\1/')

    if [ -z "$cmd" ]; then
        poyo_output "false" "" "请提供要执行的命令"
        return
    fi

    poyo_log "info" "执行命令: $cmd"

    # 安全检查
    if echo "$cmd" | grep -qE "rm\s+-rf|mkfs|dd\s+if"; then
        poyo_output "false" "" "禁止执行危险命令"
        return
    fi

    local output
    if output=$(eval "$cmd" 2>&1); then
        poyo_output "true" "{\"command\": \"$cmd\", \"output\": \"$output\"}"
    else
        poyo_output "false" "" "命令执行失败: $output"
    fi
}

# ═══════════════════════════════════════════════════════
# 🪝 Hook Implementations
# ═══════════════════════════════════════════════════════

hook_pre_tool_use() {
    local input="$1"
    local tool=$(echo "$input" | grep -o '"tool"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*: *"\([^"]*\)".*/\1/')

    poyo_log "debug" "Shell 钩子: 工具即将执行 - $tool"

    # 返回允许执行
    poyo_output "true" "{\"blocked\": false, \"message\": \"Shell 插件允许执行 $tool\"}"
}

# ═══════════════════════════════════════════════════════
# 🚀 Command Implementations
# ═══════════════════════════════════════════════════════

command_shell_demo() {
    poyo_say "欢迎来到 Shell 插件演示!"

    local result=$(cat <<EOF
{
    "message": "🐚 Shell 插件演示完成!",
    "environment": {
        "plugin_id": "$POYO_PLUGIN_ID",
        "plugin_name": "$POYO_PLUGIN_NAME",
        "dream_land": "$POYO_DREAM_LAND"
    },
    "available_tools": ["shell_info", "shell_greet", "shell_execute"],
    "system_info": {
        "os": "$(uname -s)",
        "shell": "$SHELL"
    }
}
EOF
)

    poyo_output "true" "$result"
}

# ═══════════════════════════════════════════════════════
# 📋 Main Dispatcher
# ═══════════════════════════════════════════════════════

main() {
    # 读取输入
    local input=$(poyo_get_input)
    local method=$(echo "$input" | grep -o '"method"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*: *"\([^"]*\)".*/\1/')
    local args=$(echo "$input" | grep -o '"args"[[:space:]]*:[[:space:]]*{[^}]*}' | sed 's/.*: *//')

    poyo_log "debug" "收到方法调用: $method"

    case "$method" in
        # Tools
        "shell_info"|"tool_shell_info")
            tool_shell_info "$args"
            ;;
        "shell_greet"|"tool_shell_greet")
            tool_shell_greet "$args"
            ;;
        "shell_execute"|"tool_shell_execute")
            tool_shell_execute "$args"
            ;;

        # Hooks
        "PreToolUse")
            hook_pre_tool_use "$input"
            ;;

        # Commands
        "shell_demo"|"command_shell_demo")
            command_shell_demo
            ;;

        *)
            poyo_output "false" "" "未知方法: $method"
            ;;
    esac
}

# 运行主函数
main
