#!/usr/bin/env python3
"""
🔧 Poyo Git Helper Plugin
Git 操作增强插件
"""

import json
import subprocess
import sys
import os
import re
from typing import Dict, Any, List, Optional
from datetime import datetime

# ═══════════════════════════════════════════════════════
# 🔧 Poyo API Helper
# ═══════════════════════════════════════════════════════

class PoyoAPI:
    @staticmethod
    def get_input() -> Dict[str, Any]:
        try:
            return json.loads(sys.stdin.read())
        except json.JSONDecodeError:
            return {}

    @staticmethod
    def output(result: Any, success: bool = True, error: str = None) -> None:
        output = {"success": success, "result": result}
        if error:
            output["error"] = error
        print(json.dumps(output, ensure_ascii=False, indent=2))

    @staticmethod
    def log(message: str, level: str = "info") -> None:
        print(f"[{level.upper()}] {message}", file=sys.stderr)


# ═══════════════════════════════════════════════════════
# 🔧 Git Helper Functions
# ═══════════════════════════════════════════════════════

def run_git_command(args: List[str], cwd: str = None) -> tuple:
    """执行 git 命令"""
    try:
        result = subprocess.run(
            ["git"] + args,
            capture_output=True,
            text=True,
            cwd=cwd or os.environ.get("POYO_DREAM_LAND", ".")
        )
        return result.returncode, result.stdout, result.stderr
    except Exception as e:
        return -1, "", str(e)


def git_status_enhanced(args: Dict[str, Any]) -> Dict[str, Any]:
    """增强的 git status"""
    porcelain = args.get("porcelain", False)
    show_untracked = args.get("show_untracked", True)

    cmd_args = ["status"]
    if porcelain:
        cmd_args.append("--porcelain")

    code, stdout, stderr = run_git_command(cmd_args)

    if code != 0:
        return {"error": stderr or "git status failed"}

    result = {
        "raw_output": stdout,
        "branch": None,
        "staged": [],
        "unstaged": [],
        "untracked": []
    }

    # 解析输出
    if not porcelain:
        # 解析普通格式
        lines = stdout.split("\n")
        for line in lines:
            if line.startswith("On branch "):
                result["branch"] = line.replace("On branch ", "").strip()
            elif line.startswith("Changes to be committed:"):
                # 解析已暂存的文件
                pass  # 简化处理
    else:
        # 解析 porcelain 格式
        for line in stdout.strip().split("\n"):
            if not line:
                continue
            status = line[:2]
            filename = line[3:]

            if status == "??":
                if show_untracked:
                    result["untracked"].append(filename)
            elif status[0] in "MADRC":
                result["staged"].append({"file": filename, "status": status[0]})
            elif status[1] in "MD":
                result["unstaged"].append({"file": filename, "status": status[1]})

    # 获取当前分支（如果还没获取）
    if not result["branch"]:
        code, branch, _ = run_git_command(["branch", "--show-current"])
        result["branch"] = branch.strip()

    PoyoAPI.log(f"Git status: {len(result['staged'])} staged, {len(result['unstaged'])} unstaged")

    return result


def git_commit_smart(args: Dict[str, Any]) -> Dict[str, Any]:
    """智能提交"""
    files = args.get("files", [])
    message = args.get("message")
    auto_stage = args.get("auto_stage", False)

    if auto_stage:
        # 自动暂存所有修改
        code, _, stderr = run_git_command(["add", "-A"])
        if code != 0:
            return {"error": f"Failed to stage files: {stderr}"}

    elif files:
        # 暂存指定文件
        code, _, stderr = run_git_command(["add"] + files)
        if code != 0:
            return {"error": f"Failed to stage files: {stderr}"}

    # 生成 commit message
    if not message:
        # 获取 diff 来生成智能提交信息
        code, diff, _ = run_git_command(["diff", "--cached", "--stat"])
        lines = diff.strip().split("\n")

        if len(lines) == 0:
            return {"error": "No changes to commit"}

        # 分析变更
        added = sum(1 for l in lines if "insertion" in l)
        deleted = sum(1 for l in lines if "deletion" in l)

        # 获取修改的文件类型
        code, status, _ = run_git_command(["status", "--porcelain"])
        file_types = set()
        for line in status.strip().split("\n"):
            if line:
                filename = line[3:]
                ext = os.path.splitext(filename)[1]
                if ext:
                    file_types.add(ext)

        # 生成提交信息
        changes_desc = []
        if added:
            changes_desc.append(f"+{added} additions")
        if deleted:
            changes_desc.append(f"-{deleted} deletions")

        message = f"Update {', '.join(file_types) if file_types else 'files'} ({', '.join(changes_desc)})"

    # 执行提交
    code, stdout, stderr = run_git_command(["commit", "-m", message])

    if code != 0:
        return {"error": stderr or "Commit failed"}

    # 获取提交 hash
    code, commit_hash, _ = run_git_command(["rev-parse", "HEAD"])

    PoyoAPI.log(f"Committed: {commit_hash[:8]} - {message}")

    return {
        "commit_hash": commit_hash.strip(),
        "message": message,
        "files": files if files else "all staged"
    }


def git_branch_manager(args: Dict[str, Any]) -> Dict[str, Any]:
    """分支管理"""
    action = args.get("action")
    branch = args.get("branch")
    base = args.get("base", "main")

    if action == "list":
        code, stdout, stderr = run_git_command(["branch", "-a"])
        if code != 0:
            return {"error": stderr}

        branches = []
        current = None

        for line in stdout.strip().split("\n"):
            line = line.strip()
            if line.startswith("* "):
                current = line[2:]
                branches.append({"name": current, "current": True})
            elif line and not line.startswith("remotes/"):
                branches.append({"name": line, "current": False})

        return {"branches": branches, "current": current}

    elif action == "create":
        if not branch:
            return {"error": "branch name is required"}

        code, _, stderr = run_git_command(["checkout", "-b", branch, base])
        if code != 0:
            return {"error": stderr}

        PoyoAPI.log(f"Created branch: {branch} from {base}")
        return {"created": branch, "base": base}

    elif action == "delete":
        if not branch:
            return {"error": "branch name is required"}

        code, _, stderr = run_git_command(["branch", "-D", branch])
        if code != 0:
            return {"error": stderr}

        PoyoAPI.log(f"Deleted branch: {branch}")
        return {"deleted": branch}

    elif action == "switch":
        if not branch:
            return {"error": "branch name is required"}

        code, _, stderr = run_git_command(["checkout", branch])
        if code != 0:
            return {"error": stderr}

        PoyoAPI.log(f"Switched to branch: {branch}")
        return {"switched": branch}

    else:
        return {"error": f"Unknown action: {action}"}


# ═══════════════════════════════════════════════════════
# 🪝 Hooks
# ═══════════════════════════════════════════════════════

def on_pre_tool_use(input: Dict[str, Any]) -> Dict[str, Any]:
    tool = input.get("tool", "unknown")
    PoyoAPI.log(f"Git helper tracking: {tool}", "debug")
    return {"blocked": False}


def on_post_tool_use(input: Dict[str, Any]) -> Dict[str, Any]:
    tool = input.get("tool", "unknown")
    success = input.get("success", False)

    if tool == "Bash" and success:
        # 检查是否是 git 命令
        args = input.get("args", {})
        command = args.get("command", "")

        if command.startswith("git "):
            PoyoAPI.log(f"Git command executed: {command[:50]}...")

    return {"logged": True}


# ═══════════════════════════════════════════════════════
# 🚀 Commands
# ═══════════════════════════════════════════════════════

def git_log_pretty(args: Dict[str, Any]) -> Dict[str, Any]:
    """美化的 git log"""
    cmd_args = [
        "log",
        "--oneline",
        "--graph",
        "--decorate",
        "--color=never",
        "-20"
    ]

    code, stdout, stderr = run_git_command(cmd_args)

    if code != 0:
        return {"error": stderr}

    return {
        "log": stdout,
        "format": "graph",
        "count": len(stdout.strip().split("\n"))
    }


# ═══════════════════════════════════════════════════════
# 📋 Main Dispatcher
# ═══════════════════════════════════════════════════════

def main():
    input_data = PoyoAPI.get_input()
    method = input_data.get("method", "")
    args = input_data.get("args", {})

    # 方法路由
    methods = {
        "git_status_enhanced": git_status_enhanced,
        "git_commit_smart": git_commit_smart,
        "git_branch_manager": git_branch_manager,
        "git_log_pretty": git_log_pretty,
        "PreToolUse": on_pre_tool_use,
        "PostToolUse": on_post_tool_use,
    }

    if method not in methods:
        PoyoAPI.output(None, False, f"Unknown method: {method}")
        return

    try:
        # Hooks 接收完整 input，其他接收 args
        if method in ["PreToolUse", "PostToolUse"]:
            result = methods[method](input_data)
        else:
            result = methods[method](args)
        PoyoAPI.output(result)
    except Exception as e:
        PoyoAPI.output(None, False, str(e))


if __name__ == "__main__":
    main()
