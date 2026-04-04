#!/usr/bin/env python3
"""
🐍 Poyo Python Script Plugin Example
展示 Script 插件的所有能力

环境变量说明:
- POYO_API_VERSION: API 版本
- POYO_PLUGIN_ID: 插件 ID
- POYO_PLUGIN_NAME: 插件名称
- POYO_PLUGIN_VERSION: 插件版本
- POYO_PLUGIN_PATH: 插件路径
- POYO_DREAM_LAND: 梦之国（工作目录）
- POYO_PLUGIN_CONFIG: 插件配置 (JSON)
"""

import json
import sys
import os
import time
from typing import Dict, Any, List, Optional

# ═══════════════════════════════════════════════════════
# 🔧 Poyo API Helper
# ═══════════════════════════════════════════════════════

class PoyoAPI:
    """Poyo API 辅助类"""

    @staticmethod
    def get_input() -> Dict[str, Any]:
        """从 stdin 获取输入"""
        try:
            return json.loads(sys.stdin.read())
        except json.JSONDecodeError:
            return {}

    @staticmethod
    def output(result: Any, success: bool = True, error: str = None, logs: List[str] = None) -> None:
        """输出结果到 stdout"""
        output = {
            "success": success,
            "result": result,
            "logs": logs or []
        }
        if error:
            output["error"] = error
        print(json.dumps(output, ensure_ascii=False, indent=2))

    @staticmethod
    def get_env() -> Dict[str, Any]:
        """获取插件环境"""
        config_str = os.environ.get("POYO_PLUGIN_CONFIG", "{}")
        try:
            config = json.loads(config_str)
        except json.JSONDecodeError:
            config = {}

        return {
            "api_version": os.environ.get("POYO_API_VERSION", ""),
            "plugin_id": os.environ.get("POYO_PLUGIN_ID", ""),
            "plugin_name": os.environ.get("POYO_PLUGIN_NAME", ""),
            "plugin_version": os.environ.get("POYO_PLUGIN_VERSION", ""),
            "plugin_path": os.environ.get("POYO_PLUGIN_PATH", ""),
            "dream_land": os.environ.get("POYO_DREAM_LAND", ""),
            "config": config
        }

    @staticmethod
    def log(message: str, level: str = "info") -> None:
        """日志输出到 stderr"""
        print(f"[{level.upper()}] {message}", file=sys.stderr)

    @staticmethod
    def say(message: str) -> None:
        """Poyo 说话！"""
        print(f"💚 Poyo 说: {message}", file=sys.stderr)


# ═══════════════════════════════════════════════════════
# 🎯 Tool Implementations
# ═══════════════════════════════════════════════════════

def tool_py_hello(input: Dict[str, Any]) -> Dict[str, Any]:
    """Python Hello 工具"""
    name = input.get("name", "World")
    env = PoyoAPI.get_env()

    PoyoAPI.say(f"你好, {name}!")

    return {
        "greeting": f"你好, {name}! 来自 Python 插件~",
        "plugin": env["plugin_name"],
        "version": env["plugin_version"],
        "dream_land": env["dream_land"]
    }


def tool_py_analyze(input: Dict[str, Any]) -> Dict[str, Any]:
    """文本分析工具"""
    text = input.get("text", "")

    if not text:
        return {"error": "请提供要分析的文本"}

    result = {
        "original_text": text,
        "length": len(text),
        "word_count": len(text.split()),
        "char_count": len(text),
        "line_count": len(text.split("\n")),
        "has_chinese": any('\u4e00' <= c <= '\u9fff' for c in text),
        "analysis_time": time.strftime("%Y-%m-%d %H:%M:%S")
    }

    PoyoAPI.log(f"分析了 {result['length']} 个字符")
    return result


def tool_py_calculate(input: Dict[str, Any]) -> Dict[str, Any]:
    """计算工具"""
    a = input.get("a", 0)
    b = input.get("b", 0)
    operation = input.get("operation", "add")

    operations = {
        "add": lambda x, y: x + y,
        "sub": lambda x, y: x - y,
        "mul": lambda x, y: x * y,
        "div": lambda x, y: x / y if y != 0 else None,
        "pow": lambda x, y: x ** y,
        "mod": lambda x, y: x % y if y != 0 else None,
    }

    if operation not in operations:
        return {"error": f"未知操作: {operation}"}

    try:
        result = operations[operation](a, b)
        if result is None:
            return {"error": "除数不能为零"}

        PoyoAPI.log(f"计算: {a} {operation} {b} = {result}")
        return {
            "a": a,
            "b": b,
            "operation": operation,
            "result": result
        }
    except Exception as e:
        return {"error": str(e)}


def tool_py_list_files(input: Dict[str, Any]) -> Dict[str, Any]:
    """列出文件工具"""
    path = input.get("path", ".")
    env = PoyoAPI.get_env()

    # 使用梦之国作为基础路径
    if not os.path.isabs(path):
        path = os.path.join(env["dream_land"], path)

    if not os.path.exists(path):
        return {"error": f"路径不存在: {path}"}

    files = []
    dirs = []

    try:
        for entry in os.listdir(path):
            full_path = os.path.join(path, entry)
            if os.path.isdir(full_path):
                dirs.append(entry)
            else:
                files.append(entry)

        PoyoAPI.log(f"列出了 {len(files)} 个文件和 {len(dirs)} 个目录")
        return {
            "path": path,
            "files": files,
            "directories": dirs,
            "total_files": len(files),
            "total_dirs": len(dirs)
        }
    except Exception as e:
        return {"error": str(e)}


# ═══════════════════════════════════════════════════════
# 🪝 Hook Implementations
# ═══════════════════════════════════════════════════════

def hook_pre_tool_use(input: Dict[str, Any]) -> Dict[str, Any]:
    """PreToolUse 钩子"""
    tool = input.get("tool", "unknown")
    PoyoAPI.log(f"工具即将执行: {tool}", "debug")

    # 示例：阻止某些危险操作
    # if tool == "Bash":
    #     args = input.get("args", {})
    #     command = args.get("command", "")
    #     if "rm -rf" in command:
    #         return {
    #             "blocked": True,
    #             "reason": "禁止执行 rm -rf 命令"
    #         }

    return {
        "blocked": False,
        "message": f"Python 插件允许执行 {tool}"
    }


def hook_post_tool_use(input: Dict[str, Any]) -> Dict[str, Any]:
    """PostToolUse 钩子"""
    tool = input.get("tool", "unknown")
    success = input.get("success", False)

    if success:
        PoyoAPI.log(f"工具 {tool} 执行成功")
    else:
        PoyoAPI.log(f"工具 {tool} 执行失败: {input.get('error', '未知错误')}", "warn")

    return {"logged": True}


# ═══════════════════════════════════════════════════════
# 🚀 Command Implementations
# ═══════════════════════════════════════════════════════

def command_py_demo(input: Dict[str, Any]) -> Dict[str, Any]:
    """Python 插件演示命令"""
    env = PoyoAPI.get_env()

    PoyoAPI.say("欢迎来到 Python 插件演示!")
    PoyoAPI.log("执行演示命令")

    return {
        "message": "🐍 Python 插件演示完成!",
        "environment": {
            "plugin_id": env["plugin_id"],
            "plugin_name": env["plugin_name"],
            "dream_land": env["dream_land"]
        },
        "available_tools": ["py_hello", "py_analyze", "py_calculate", "py_list_files"],
        "available_hooks": ["PreToolUse", "PostToolUse"],
        "config": env["config"]
    }


# ═══════════════════════════════════════════════════════
# 📋 Main Dispatcher
# ═══════════════════════════════════════════════════════

def main():
    """主入口"""
    # 获取输入
    input_data = PoyoAPI.get_input()
    method = input_data.get("method", "")
    args = input_data.get("args", {})

    # 方法路由
    methods = {
        # Tools
        "py_hello": tool_py_hello,
        "tool_py_hello": tool_py_hello,
        "py_analyze": tool_py_analyze,
        "tool_py_analyze": tool_py_analyze,
        "py_calculate": tool_py_calculate,
        "tool_py_calculate": tool_py_calculate,
        "py_list_files": tool_py_list_files,
        "tool_py_list_files": tool_py_list_files,

        # Hooks
        "PreToolUse": hook_pre_tool_use,
        "PostToolUse": hook_post_tool_use,

        # Commands
        "py_demo": command_py_demo,
        "command_py_demo": command_py_demo,
    }

    if method not in methods:
        PoyoAPI.output(
            None,
            success=False,
            error=f"未知方法: {method}",
            logs=[f"可用方法: {list(methods.keys())}"]
        )
        return

    try:
        result = methods[method](args)
        PoyoAPI.output(result)
    except Exception as e:
        PoyoAPI.output(None, success=False, error=str(e))


if __name__ == "__main__":
    main()
