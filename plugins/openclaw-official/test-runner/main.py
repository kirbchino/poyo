#!/usr/bin/env python3
"""
🧪 OpenClaw Test Runner Plugin
测试执行和管理工具 - OpenClaw 格式示例
"""

import json
import os
import subprocess
import sys
import re
import time
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
        output = {
            "success": success,
            "result": result,
            "timestamp": datetime.now().isoformat()
        }
        if error:
            output["error"] = error
        print(json.dumps(output, ensure_ascii=False, indent=2))

    @staticmethod
    def get_env() -> Dict[str, Any]:
        return {
            "pluginId": os.environ.get("POYO_PLUGIN_ID", "test-runner"),
            "dreamLand": os.environ.get("POYO_DREAM_LAND", os.getcwd()),
            "config": json.loads(os.environ.get("POYO_PLUGIN_CONFIG", "{}"))
        }

    @staticmethod
    def log(message: str, level: str = "info") -> None:
        print(f"[{level.upper()}] [TestRunner] {message}", file=sys.stderr)


# ═══════════════════════════════════════════════════════
# 🎯 Tool Implementations
# ═══════════════════════════════════════════════════════

def run_tests(args: Dict[str, Any]) -> Dict[str, Any]:
    """运行测试套件"""
    test_path = args.get("path")
    framework = args.get("framework", "pytest")
    coverage = args.get("coverage", False)
    parallel = args.get("parallel", 1)
    env = PoyoAPI.get_env()

    if not test_path:
        return {"error": "path is required"}

    # 解析路径
    full_path = test_path if os.path.isabs(test_path) else os.path.join(env["dreamLand"], test_path)

    if not os.path.exists(full_path):
        return {"error": f"Path not found: {full_path}"}

    PoyoAPI.log(f"Running tests: {full_path} with {framework}")

    start_time = time.time()

    # 构建测试命令
    commands = {
        "pytest": ["pytest", "-v", "--tb=short"],
        "jest": ["npx", "jest", "--verbose"],
        "go-test": ["go", "test", "-v", "./..."],
        "unittest": ["python", "-m", "unittest", "discover", "-v"]
    }

    if framework not in commands:
        return {"error": f"Unsupported framework: {framework}"}

    cmd = commands[framework].copy()

    # 添加路径
    if framework in ["pytest", "jest"]:
        cmd.append(full_path)
    elif framework == "unittest":
        cmd.extend(["-s", full_path])

    # 覆盖率选项
    if coverage and framework == "pytest":
        cmd.extend(["--cov=.", "--cov-report=term-missing"])
    elif coverage and framework == "jest":
        cmd.append("--coverage")

    # 并行执行
    if parallel > 1:
        if framework == "pytest":
            cmd.extend(["-n", str(parallel)])
        elif framework == "jest":
            cmd.extend(["--maxWorkers", str(parallel)])

    # 执行测试
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=env["dreamLand"],
            timeout=env["config"].get("timeout", 300)
        )

        duration = time.time() - start_time

        # 解析结果
        output = result.stdout + result.stderr
        test_results = parse_test_output(output, framework)

        PoyoAPI.log(f"Tests completed: {test_results.get('passed', 0)} passed, {test_results.get('failed', 0)} failed")

        return {
            "path": test_path,
            "framework": framework,
            "duration": round(duration, 2),
            "return_code": result.returncode,
            "results": test_results,
            "output": output,
            "success": result.returncode == 0
        }

    except subprocess.TimeoutExpired:
        return {"error": "Test execution timed out"}
    except Exception as e:
        return {"error": str(e)}


def parse_test_output(output: str, framework: str) -> Dict[str, Any]:
    """解析测试输出"""
    results = {
        "total": 0,
        "passed": 0,
        "failed": 0,
        "skipped": 0,
        "errors": 0,
        "failures": []
    }

    if framework == "pytest":
        # 解析 pytest 输出
        match = re.search(r"(\d+) passed", output)
        if match:
            results["passed"] = int(match.group(1))

        match = re.search(r"(\d+) failed", output)
        if match:
            results["failed"] = int(match.group(1))

        match = re.search(r"(\d+) skipped", output)
        if match:
            results["skipped"] = int(match.group(1))

        match = re.search(r"(\d+) error", output)
        if match:
            results["errors"] = int(match.group(1))

    elif framework == "jest":
        # 解析 jest 输出
        match = re.search(r"Tests:\s+(\d+) passed", output)
        if match:
            results["passed"] = int(match.group(1))

        match = re.search(r"(\d+) failed", output)
        if match:
            results["failed"] = int(match.group(1))

    elif framework == "go-test":
        # 解析 go test 输出
        match = re.search(r"PASS\s+(\d+)", output)
        if match:
            results["passed"] = int(match.group(1))

        match = re.search(r"FAIL\s+(\d+)", output)
        if match:
            results["failed"] = int(match.group(1))

    results["total"] = results["passed"] + results["failed"] + results["skipped"] + results["errors"]
    return results


def list_tests(args: Dict[str, Any]) -> Dict[str, Any]:
    """列出所有可用的测试"""
    test_path = args.get("path")
    pattern = args.get("pattern", "")
    env = PoyoAPI.get_env()

    if not test_path:
        return {"error": "path is required"}

    full_path = test_path if os.path.isabs(test_path) else os.path.join(env["dreamLand"], test_path)

    if not os.path.exists(full_path):
        return {"error": f"Path not found: {full_path}"}

    PoyoAPI.log(f"Listing tests in: {full_path}")

    tests = []

    # 扫描测试文件
    for root, dirs, files in os.walk(full_path):
        # 跳过常见排除目录
        dirs[:] = [d for d in dirs if d not in ["node_modules", ".git", "__pycache__", "venv"]]

        for file in files:
            # 检测测试文件
            is_test = False
            if file.startswith("test_") or file.endswith("_test.py"):
                is_test = True
            elif file.startswith("test") and file.endswith((".js", ".ts")):
                is_test = True
            elif file.endswith("_test.go"):
                is_test = True

            if is_test:
                file_path = os.path.join(root, file)
                rel_path = os.path.relpath(file_path, full_path)

                # 应用过滤模式
                if pattern and pattern.lower() not in rel_path.lower():
                    continue

                tests.append({
                    "path": rel_path,
                    "absolute_path": file_path,
                    "type": os.path.splitext(file)[1]
                })

    PoyoAPI.log(f"Found {len(tests)} test files")

    return {
        "path": test_path,
        "tests": tests,
        "total": len(tests),
        "pattern": pattern
    }


def generate_test(args: Dict[str, Any]) -> Dict[str, Any]:
    """为代码自动生成测试"""
    source_file = args.get("source_file")
    output_dir = args.get("output_dir", "./tests")
    style = args.get("style", "unit")
    env = PoyoAPI.get_env()

    if not source_file:
        return {"error": "source_file is required"}

    full_path = source_file if os.path.isabs(source_file) else os.path.join(env["dreamLand"], source_file)

    if not os.path.exists(full_path):
        return {"error": f"Source file not found: {full_path}"}

    PoyoAPI.log(f"Generating tests for: {full_path}")

    # 读取源文件
    with open(full_path, 'r') as f:
        source_content = f.read()

    # 解析函数和类
    functions = re.findall(r'def\s+(\w+)\s*\([^)]*\)', source_content)
    classes = re.findall(r'class\s+(\w+)', source_content)

    # 确定测试框架
    ext = os.path.splitext(source_file)[1]
    framework = {
        '.py': 'pytest',
        '.js': 'jest',
        '.ts': 'jest',
        '.go': 'go-test'
    }.get(ext, 'pytest')

    # 生成测试内容
    test_content = generate_test_content(framework, functions, classes, source_file, style)

    # 确保输出目录存在
    full_output_dir = output_dir if os.path.isabs(output_dir) else os.path.join(env["dreamLand"], output_dir)
    os.makedirs(full_output_dir, exist_ok=True)

    # 生成测试文件名
    test_filename = generate_test_filename(source_file, framework)
    test_file_path = os.path.join(full_output_dir, test_filename)

    # 写入测试文件
    with open(test_file_path, 'w') as f:
        f.write(test_content)

    PoyoAPI.log(f"Generated test file: {test_file_path}")

    return {
        "source_file": source_file,
        "test_file": test_file_path,
        "framework": framework,
        "functions_tested": len(functions),
        "classes_tested": len(classes),
        "style": style
    }


def generate_test_content(framework: str, functions: List[str], classes: List[str],
                          source_file: str, style: str) -> str:
    """生成测试内容"""
    if framework == 'pytest':
        lines = [
            '"""',
            f'Generated tests for {source_file}',
            f'Style: {style}',
            '"""',
            'import pytest',
            f'from {os.path.splitext(os.path.basename(source_file))[0]} import *',
            '',
        ]

        for func in functions:
            if not func.startswith('_'):
                lines.extend([
                    f'def test_{func}():',
                    f'    """Test {func}"""',
                    f'    # TODO: Implement test',
                    f'    pass',
                    ''
                ])

        for cls in classes:
            lines.extend([
                f'class Test{cls}:',
                f'    """Tests for {cls}"""',
                '',
                f'    def test_initialization(self):',
                f'        """Test {cls} initialization"""',
                f'        # TODO: Implement test',
                f'        pass',
                ''
            ])

        return '\n'.join(lines)

    elif framework == 'jest':
        lines = [
            '/**',
            f' * Generated tests for {source_file}',
            f' * Style: {style}',
            ' */',
            '',
            f'const {{ {", ".join(functions[:5])} }} = require("./{os.path.splitext(os.path.basename(source_file))[0]}");',
            '',
            'describe("Generated Tests", () => {',
        ]

        for func in functions:
            if not func.startswith('_'):
                lines.extend([
                    f'    test("{func} should work correctly", () => {{',
                    f'        // TODO: Implement test',
                    f'        expect(true).toBe(true);',
                    f'    }});',
                    ''
                ])

        lines.append('});')
        return '\n'.join(lines)

    elif framework == 'go-test':
        lines = [
            f'// Generated tests for {source_file}',
            f'// Style: {style}',
            'package main',
            '',
            'import "testing"',
            '',
        ]

        for func in functions:
            if func[0].isupper():  # exported functions
                lines.extend([
                    f'func Test{func.capitalize()}(t *testing.T) {{',
                    f'    // TODO: Implement test',
                    f'}}',
                    ''
                ])

        return '\n'.join(lines)

    return f'# Generated tests for {source_file}'


def generate_test_filename(source_file: str, framework: str) -> str:
    """生成测试文件名"""
    base = os.path.splitext(os.path.basename(source_file))[0]

    if framework == 'pytest':
        return f'test_{base}.py'
    elif framework == 'jest':
        return f'{base}.test.js'
    elif framework == 'go-test':
        return f'{base}_test.go'
    else:
        return f'test_{base}'


# ═══════════════════════════════════════════════════════
# 🪝 Hooks
# ═══════════════════════════════════════════════════════

def after_tool_use(input: Dict[str, Any]) -> Dict[str, Any]:
    """工具执行后钩子"""
    tool = input.get("tool", "unknown")
    success = input.get("success", False)

    if success:
        PoyoAPI.log(f"Tool {tool} executed successfully")
    else:
        PoyoAPI.log(f"Tool {tool} failed: {input.get('error', 'unknown error')}", "warn")

    return {"logged": True, "source": "test-runner"}


# ═══════════════════════════════════════════════════════
# 🚀 Commands
# ═══════════════════════════════════════════════════════

def test_coverage(args: Dict[str, Any]) -> Dict[str, Any]:
    """运行测试并生成覆盖率报告"""
    path = args.get("path", ".")
    return run_tests({"path": path, "coverage": True})


# ═══════════════════════════════════════════════════════
# 📋 Main Dispatcher
# ═══════════════════════════════════════════════════════

def main():
    input_data = PoyoAPI.get_input()
    method = input_data.get("method", "")
    args = input_data.get("args", {})

    # OpenClaw 方法路由
    methods = {
        # Tools
        "run_tests": run_tests,
        "list_tests": list_tests,
        "generate_test": generate_test,

        # Hooks (OpenClaw 事件格式)
        "tool.use.after": after_tool_use,

        # Commands
        "test-coverage": test_coverage,
    }

    if method not in methods:
        PoyoAPI.output(None, False, f"Unknown method: {method}")
        return

    try:
        # Hooks 接收完整 input
        if "tool.use" in method:
            result = methods[method](input_data)
        else:
            result = methods[method](args)
        PoyoAPI.output(result)
    except Exception as e:
        PoyoAPI.output(None, False, str(e))


if __name__ == "__main__":
    main()
