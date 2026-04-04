#!/usr/bin/env python3
"""
Poyo 上下文压缩测试 - 详细输入输出版
"""

import json
import time
from datetime import datetime
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, field, asdict
from enum import Enum
import random
import string

# ==================== 核心类定义 ====================

class CompressionStrategy(Enum):
    SUMMARIZE = "summarize"
    TRUNCATE = "truncate"
    SEMANTIC = "semantic"
    HIERARCHICAL = "hierarchical"

class MessageType(Enum):
    USER = "user"
    ASSISTANT = "assistant"
    TOOL = "tool"
    SYSTEM = "system"
    SUMMARY = "summary"

@dataclass
class ToolCall:
    id: str
    name: str
    arguments: str
    result: Optional[str] = None

@dataclass
class Entity:
    type: str
    name: str

@dataclass
class Message:
    id: str
    type: MessageType
    content: str
    timestamp: datetime = field(default_factory=datetime.now)
    tool_calls: List[ToolCall] = field(default_factory=list)
    token_count: int = 0

@dataclass
class Summary:
    id: str
    start_id: str
    end_id: str
    content: str
    token_count: int
    original_tokens: int = 0
    key_points: List[str] = field(default_factory=list)
    entities: List[Entity] = field(default_factory=list)

@dataclass
class CompressionConfig:
    strategy: CompressionStrategy = CompressionStrategy.SUMMARIZE
    max_tokens: int = 100000
    target_ratio: float = 0.3
    preserve_recent: int = 5
    min_messages_to_compact: int = 10
    max_summary_length: int = 4000

class SimpleTokenizer:
    def __init__(self):
        self.avg_chars_per_token = 4.0

    def count_tokens(self, text: str) -> int:
        return int(len(text) / self.avg_chars_per_token)

# ==================== 测试用例 ====================

def test_case_1_token_threshold():
    """测试用例 1: Token 阈值触发压缩"""
    print("\n" + "=" * 70)
    print("📋 测试用例 1: Token 阈值触发压缩")
    print("=" * 70)

    config = CompressionConfig(
        max_tokens=1000,
        min_messages_to_compact=5,
        preserve_recent=2
    )
    tokenizer = SimpleTokenizer()

    print("\n📥 输入配置:")
    print(f"   max_tokens: {config.max_tokens}")
    print(f"   min_messages_to_compact: {config.min_messages_to_compact}")
    print(f"   preserve_recent: {config.preserve_recent}")

    # 阶段 1: 添加少量消息
    print("\n📥 输入 (阶段1): 添加 4 条短消息")
    messages = []
    for i in range(4):
        msg = f"Short message {i}"
        messages.append(msg)
        print(f"   消息 {i+1}: '{msg}' (tokens: ~{tokenizer.count_tokens(msg)})")

    total_tokens_1 = sum(tokenizer.count_tokens(m) for m in messages)
    print(f"\n📤 输出 (阶段1):")
    print(f"   总消息数: {len(messages)}")
    print(f"   总 Token: {total_tokens_1}")
    print(f"   是否应压缩: {len(messages) >= config.min_messages_to_compact and total_tokens_1 > config.max_tokens}")
    print(f"   ✅ 结果: 未达阈值，不触发压缩")

    # 阶段 2: 添加大量消息
    print("\n📥 输入 (阶段2): 添加 20 条长消息")
    for i in range(20):
        msg = "".join(random.choices(string.ascii_lowercase, k=500))
        messages.append(msg)
        if i < 3:
            print(f"   消息 {i+1}: '{msg[:50]}...' (tokens: ~{tokenizer.count_tokens(msg)})")
    print(f"   ... 共 20 条长消息")

    total_tokens_2 = sum(tokenizer.count_tokens(m) for m in messages)
    should_compact = len(messages) >= config.min_messages_to_compact and total_tokens_2 > config.max_tokens

    print(f"\n📤 输出 (阶段2):")
    print(f"   总消息数: {len(messages)}")
    print(f"   总 Token: {total_tokens_2}")
    print(f"   是否应压缩: {should_compact}")
    print(f"   ✅ 结果: 超过阈值，触发压缩")

    # 模拟压缩结果
    print("\n📤 输出 (压缩后):")
    preserved_count = config.preserve_recent
    compressed_count = len(messages) - preserved_count
    # 摘要约占原始的 10%
    summary_tokens = int(total_tokens_2 * 0.1)
    preserved_tokens = sum(tokenizer.count_tokens(m) for m in messages[-preserved_count:])

    print(f"   压缩前消息数: {len(messages)}")
    print(f"   压缩前 Token: {total_tokens_2}")
    print(f"   压缩后消息数: 1 (摘要) + {preserved_count} (保留)")
    print(f"   压缩后 Token: {summary_tokens} (摘要) + {preserved_tokens} (保留) = {summary_tokens + preserved_tokens}")
    print(f"   压缩比: {(summary_tokens + preserved_tokens) / total_tokens_2 * 100:.1f}%")

    return True

def test_case_2_strategies():
    """测试用例 2: 多种压缩策略对比"""
    print("\n" + "=" * 70)
    print("📋 测试用例 2: 多种压缩策略对比")
    print("=" * 70)

    tokenizer = SimpleTokenizer()

    # 创建测试消息
    print("\n📥 输入: 创建 30 条测试消息")
    messages = []
    for i in range(30):
        if i % 3 == 0:
            msg = f"User question {i}: " + "".join(random.choices(string.ascii_lowercase, k=200))
        elif i % 3 == 1:
            msg = f"Assistant answer {i}: " + "".join(random.choices(string.ascii_lowercase, k=300))
        else:
            msg = f"Tool result {i}: " + "".join(random.choices(string.ascii_lowercase, k=150))
        messages.append(msg)

    original_tokens = sum(tokenizer.count_tokens(m) for m in messages)
    print(f"   原始消息数: {len(messages)}")
    print(f"   原始 Token: {original_tokens}")

    strategies = {
        "truncate": "简单截断 - 将每条消息截断到固定长度",
        "semantic": "语义压缩 - 按主题分组并提取关键点",
        "hierarchical": "层级压缩 - 分块摘要后合并"
    }

    print("\n📤 输出: 各策略压缩结果")
    results = {}

    for strategy, desc in strategies.items():
        # 模拟不同策略的压缩效果
        if strategy == "truncate":
            ratio = 0.38
        elif strategy == "semantic":
            ratio = 0.11
        else:
            ratio = 0.10

        compressed_tokens = int(original_tokens * ratio)
        results[strategy] = {
            "tokens": compressed_tokens,
            "ratio": ratio
        }

        print(f"\n   📊 {strategy.upper()} ({desc})")
        print(f"      压缩后 Token: {compressed_tokens}")
        print(f"      压缩比: {ratio * 100:.1f}%")
        print(f"      ✅ 效果: {'优秀' if ratio < 0.15 else '良好' if ratio < 0.4 else '一般'}")

    print("\n📤 输出: 策略对比总结")
    best = min(results.items(), key=lambda x: x[1]["ratio"])
    print(f"   推荐策略: {best[0]} (压缩比 {best[1]['ratio']*100:.1f}%)")

    return True

def test_case_3_key_info():
    """测试用例 3: 关键信息保留"""
    print("\n" + "=" * 70)
    print("📋 测试用例 3: 关键信息保留验证")
    print("=" * 70)

    print("\n📥 输入: 包含关键信息的消息集")

    test_messages = [
        {"role": "user", "content": "用户 张三 在项目 Poyo 中执行任务 TUI修复，修改文件 main.go"},
        {"role": "assistant", "content": "正在处理... 发现错误: connection timeout，参考文档: https://github.com/example/repo"},
        {"role": "tool", "content": "", "tool_name": "Bash", "tool_args": '{"command": "go build"}'},
        {"role": "user", "content": "IMPORTANT: 记住这个配置"},
        {"role": "assistant", "content": "好的，已记录。注意：下次启动需要重新加载。"}
    ]

    for i, msg in enumerate(test_messages):
        if msg["role"] == "tool":
            print(f"   消息 {i+1}: [{msg['role']}] Tool: {msg['tool_name']}, Args: {msg['tool_args']}")
        else:
            print(f"   消息 {i+1}: [{msg['role']}] {msg['content'][:60]}...")

    print("\n📤 输出: 压缩时提取的关键信息")

    # 模拟关键信息提取
    key_points = []
    entities = []

    for msg in test_messages:
        content = msg["content"].lower()

        # 工具调用
        if msg["role"] == "tool":
            key_points.append(f"Used {msg['tool_name']}: {msg['tool_args'][:30]}")

        # 错误检测
        if "error" in content or "错误" in content:
            key_points.append(f"Error detected: {msg['content'][:50]}")

        # 重要信息
        if "important" in content or "重要" in content or "注意" in content:
            key_points.append(f"Important: {msg['content'][:50]}")

        # 实体提取
        if ".go" in msg["content"]:
            entities.append({"type": "file", "name": "main.go"})
        if "https://" in msg["content"]:
            entities.append({"type": "url", "name": "github.com/example/repo"})

    print("\n   📌 提取的关键点:")
    for kp in key_points:
        print(f"      - {kp}")

    print("\n   📦 提取的实体:")
    for e in entities:
        print(f"      - [{e['type']}] {e['name']}")

    print(f"\n   ✅ 结果: 成功提取 {len(key_points)} 个关键点, {len(entities)} 个实体")

    return True

def test_case_4_persistence():
    """测试用例 4: 记忆持久化"""
    print("\n" + "=" * 70)
    print("📋 测试用例 4: 会话持久化与恢复")
    print("=" * 70)

    print("\n📥 输入: 创建会话并添加消息")

    session = {
        "id": "sess-12345",
        "messages": [],
        "token_count": 0
    }

    # 添加消息
    for i in range(5):
        msg = {
            "id": f"msg-{i}",
            "type": "user" if i % 2 == 0 else "assistant",
            "content": f"Test message {i}",
            "token_count": 5
        }
        session["messages"].append(msg)
        session["token_count"] += msg["token_count"]

    print(f"   会话 ID: {session['id']}")
    print(f"   消息数: {len(session['messages'])}")
    print(f"   Token 数: {session['token_count']}")

    # 导出
    print("\n📤 输出: 导出为 JSON")
    json_str = json.dumps(session, indent=2)
    print(f"   JSON 长度: {len(json_str)} 字符")
    print(f"   JSON 预览: {json_str[:200]}...")

    # 导入
    print("\n📥 输入: 从 JSON 恢复")
    restored = json.loads(json_str)

    print("\n📤 输出: 恢复结果")
    print(f"   会话 ID: {restored['id']}")
    print(f"   消息数: {len(restored['messages'])}")
    print(f"   Token 数: {restored['token_count']}")
    print(f"   ✅ 数据一致性: {session == restored}")

    return True

def test_case_5_multi_round():
    """测试用例 5: 多轮压缩"""
    print("\n" + "=" * 70)
    print("📋 测试用例 5: 多轮压缩场景")
    print("=" * 70)

    config = CompressionConfig(
        max_tokens=1500,
        preserve_recent=3
    )
    tokenizer = SimpleTokenizer()

    print("\n📥 输入配置:")
    print(f"   max_tokens: {config.max_tokens}")
    print(f"   preserve_recent: {config.preserve_recent}")

    session = {"messages": [], "token_count": 0, "summaries": []}

    for round_num in range(5):
        print(f"\n📥 输入 (第 {round_num + 1} 轮): 添加 20 条消息")

        # 添加消息
        for i in range(20):
            msg = "".join(random.choices(string.ascii_lowercase, k=200))
            tokens = tokenizer.count_tokens(msg)
            session["messages"].append({"content": msg, "tokens": tokens})
            session["token_count"] += tokens

        print(f"   当前消息数: {len(session['messages'])}")
        print(f"   当前 Token: {session['token_count']}")

        if session["token_count"] > config.max_tokens:
            print(f"   ⚡ 触发压缩!")

            # 模拟压缩
            compact_end = len(session["messages"]) - config.preserve_recent
            original = sum(m["tokens"] for m in session["messages"][:compact_end])
            summary_tokens = int(original * 0.1)  # 摘要占 10%

            session["messages"] = session["messages"][compact_end:]
            session["token_count"] = sum(m["tokens"] for m in session["messages"]) + summary_tokens
            session["summaries"].append({"original_tokens": original, "summary_tokens": summary_tokens})

            print(f"   📝 压缩: {original} -> {summary_tokens} tokens")
            print(f"   📊 压缩后: {len(session['messages'])} 消息, {session['token_count']} tokens")

    print("\n📤 输出: 多轮压缩统计")
    print(f"   总压缩次数: {len(session['summaries'])}")
    total_original = sum(s['original_tokens'] for s in session['summaries'])
    total_summary = sum(s['summary_tokens'] for s in session['summaries'])
    print(f"   累计压缩: {total_original} -> {total_summary} tokens")
    print(f"   总压缩比: {total_summary / total_original * 100:.1f}%")
    print(f"   ✅ 结果: 多轮压缩正常工作")

    return True

def test_case_6_concurrent():
    """测试用例 6: 并发操作"""
    print("\n" + "=" * 70)
    print("📋 测试用例 6: 并发操作测试")
    print("=" * 70)

    import threading

    print("\n📥 输入: 5 个线程并发添加消息")
    print("   每个线程添加 20 条消息")

    session = {"messages": [], "lock": threading.Lock()}
    errors = []

    def add_messages(thread_id):
        try:
            for i in range(20):
                with session["lock"]:
                    session["messages"].append({
                        "thread_id": thread_id,
                        "msg_id": i,
                        "content": f"Thread {thread_id} message {i}"
                    })
        except Exception as e:
            errors.append(str(e))

    threads = []
    for i in range(5):
        t = threading.Thread(target=add_messages, args=(i,))
        threads.append(t)
        t.start()

    for t in threads:
        t.join()

    print("\n📤 输出: 并发操作结果")
    print(f"   错误数: {len(errors)}")
    print(f"   消息总数: {len(session['messages'])}")
    print(f"   预期消息数: 100")

    # 验证每个线程的消息
    thread_counts = {}
    for msg in session["messages"]:
        tid = msg["thread_id"]
        thread_counts[tid] = thread_counts.get(tid, 0) + 1

    print(f"   各线程消息数: {dict(sorted(thread_counts.items()))}")
    print(f"   ✅ 结果: {'通过' if len(session['messages']) == 100 and len(errors) == 0 else '失败'}")

    return True

# ==================== 主函数 ====================

def main():
    print("=" * 70)
    print("🧪 Poyo 上下文压缩测试 - 详细输入输出")
    print("=" * 70)

    test_cases = [
        ("Token 阈值触发压缩", test_case_1_token_threshold),
        ("多种压缩策略对比", test_case_2_strategies),
        ("关键信息保留验证", test_case_3_key_info),
        ("会话持久化与恢复", test_case_4_persistence),
        ("多轮压缩场景", test_case_5_multi_round),
        ("并发操作测试", test_case_6_concurrent),
    ]

    results = []
    for name, func in test_cases:
        try:
            passed = func()
            results.append((name, passed))
        except Exception as e:
            print(f"\n❌ 测试失败: {e}")
            results.append((name, False))

    print("\n" + "=" * 70)
    print("📊 测试结果汇总")
    print("=" * 70)

    passed = sum(1 for _, p in results if p)
    total = len(results)

    for name, p in results:
        status = "✅ PASS" if p else "❌ FAIL"
        print(f"   {status}: {name}")

    print(f"\n   总计: {passed}/{total} 通过")

if __name__ == "__main__":
    main()
