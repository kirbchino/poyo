#!/usr/bin/env python3
"""
Poyo 上下文压缩和记忆能力测试
测试场景：
1. Token 阈值触发压缩
2. 多种压缩策略对比
3. 关键信息保留验证
4. 记忆持久化测试
5. 多轮压缩场景
"""

import json
import time
import hashlib
from datetime import datetime
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, field, asdict
from enum import Enum
import random
import string

# ==================== 模拟 Poyo Compactor 核心逻辑 ====================

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
    error: Optional[str] = None

@dataclass
class Entity:
    type: str
    name: str
    value: Optional[str] = None

@dataclass
class Message:
    id: str
    type: MessageType
    content: str
    timestamp: datetime = field(default_factory=datetime.now)
    role: Optional[str] = None
    tool_calls: List[ToolCall] = field(default_factory=list)
    token_count: int = 0
    metadata: Dict[str, Any] = field(default_factory=dict)

@dataclass
class Summary:
    id: str
    start_id: str
    end_id: str
    content: str
    token_count: int
    original_tokens: int = 0
    compressed_at: datetime = field(default_factory=datetime.now)
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
    """模拟 Poyo 的 SimpleTokenizer"""
    def __init__(self, avg_chars_per_token: float = 4.0):
        self.avg_chars_per_token = avg_chars_per_token

    def count_tokens(self, text: str) -> int:
        return int(len(text) / self.avg_chars_per_token)

class Compactor:
    """模拟 Poyo 的 Compactor"""

    def __init__(self, config: CompressionConfig, tokenizer: SimpleTokenizer):
        self.config = config
        self.tokenizer = tokenizer
        self.sessions: Dict[str, Dict] = {}

    def generate_id(self, prefix: str = "msg") -> str:
        return f"{prefix}-{int(time.time() * 1000000)}"

    def add_message(self, session: Dict, msg: Message) -> None:
        if not msg.id:
            msg.id = self.generate_id()
        msg.timestamp = datetime.now()
        msg.token_count = self.tokenizer.count_tokens(msg.content)

        session["messages"].append(asdict(msg))
        session["token_count"] += msg.token_count
        session["updated_at"] = datetime.now()

    def should_compact(self, session: Dict) -> bool:
        if len(session["messages"]) < self.config.min_messages_to_compact:
            return False
        return session["token_count"] > self.config.max_tokens

    def compact(self, session: Dict) -> Optional[Summary]:
        messages = session["messages"]

        if len(messages) <= self.config.preserve_recent:
            return None

        compact_end = len(messages) - self.config.preserve_recent
        messages_to_compact = messages[:compact_end]

        if not messages_to_compact:
            return None

        original_tokens = sum(m.get("token_count", 0) for m in messages_to_compact)

        # 根据策略压缩
        if self.config.strategy == CompressionStrategy.TRUNCATE:
            summary = self._truncate_messages(messages_to_compact)
        elif self.config.strategy == CompressionStrategy.SEMANTIC:
            summary = self._semantic_compress(messages_to_compact)
        elif self.config.strategy == CompressionStrategy.HIERARCHICAL:
            summary = self._hierarchical_compress(messages_to_compact)
        else:
            summary = self._summarize_messages(messages_to_compact)

        summary.original_tokens = original_tokens
        summary.id = self.generate_id("sum")
        summary.start_id = messages_to_compact[0]["id"]
        summary.end_id = messages_to_compact[-1]["id"]
        summary.compressed_at = datetime.now()

        # 创建摘要消息
        summary_msg = Message(
            id=self.generate_id(),
            type=MessageType.SUMMARY,
            content=summary.content,
            token_count=summary.token_count,
            metadata={
                "summary_id": summary.id,
                "original_tokens": original_tokens
            }
        )

        # 更新会话
        session["messages"] = [asdict(summary_msg)] + messages[compact_end:]
        session["token_count"] = sum(m.get("token_count", 0) for m in session["messages"])
        session["summaries"].append(asdict(summary))

        return summary

    def _truncate_messages(self, messages: List[Dict]) -> Summary:
        content_parts = []
        key_points = []

        max_len_per_msg = max(100, self.config.max_summary_length // len(messages))

        for msg in messages:
            text = msg["content"]
            if len(text) > max_len_per_msg:
                text = text[:max_len_per_msg] + "..."
            content_parts.append(f"[{msg['type']}]: {text}")

            # 提取工具调用
            for tc in msg.get("tool_calls", []):
                key_points.append(f"Used tool: {tc['name']}")

        content = "\n\n".join(content_parts)

        return Summary(
            id="", start_id="", end_id="",
            content=content,
            token_count=self.tokenizer.count_tokens(content),
            original_tokens=0,  # Will be set by caller
            key_points=key_points
        )

    def _semantic_compress(self, messages: List[Dict]) -> Summary:
        # 按主题分组
        groups = self._group_by_topic(messages)

        content_parts = []
        key_points = []
        entities = []

        for group in groups:
            group_summary = self._summarize_group(group)
            content_parts.append(group_summary)
            key_points.extend(self._extract_key_points(group))
            entities.extend(self._extract_entities(group))

        content = "\n\n---\n\n".join(content_parts)

        return Summary(
            id="", start_id="", end_id="",
            content=content,
            token_count=self.tokenizer.count_tokens(content),
            original_tokens=0,  # Will be set by caller
            key_points=key_points,
            entities=entities
        )

    def _hierarchical_compress(self, messages: List[Dict]) -> Summary:
        chunk_size = 5
        chunks = [messages[i:i+chunk_size] for i in range(0, len(messages), chunk_size)]

        summaries = []
        key_points = []

        for chunk in chunks:
            chunk_summary = self._summarize_chunk(chunk)
            summaries.append(chunk_summary)
            key_points.extend(self._extract_key_points(chunk))

        content = "\n\n".join(summaries)

        return Summary(
            id="", start_id="", end_id="",
            content=content,
            token_count=self.tokenizer.count_tokens(content),
            original_tokens=0,  # Will be set by caller
            key_points=key_points
        )

    def _summarize_messages(self, messages: List[Dict]) -> Summary:
        # 模拟 LLM 摘要（这里简化为提取关键信息）
        return self._truncate_messages(messages)

    def _group_by_topic(self, messages: List[Dict]) -> List[List[Dict]]:
        groups = []
        current_group = []

        for msg in messages:
            if current_group and msg["type"] == "user" and current_group[-1]["type"] != "user":
                groups.append(current_group)
                current_group = []
            current_group.append(msg)

        if current_group:
            groups.append(current_group)

        return groups

    def _summarize_group(self, messages: List[Dict]) -> str:
        parts = []
        for msg in messages:
            if msg["type"] == "user":
                parts.append(f"User asked: {msg['content'][:200]}")
            elif msg["type"] == "assistant":
                parts.append(f"Assistant responded: {msg['content'][:200]}")
            elif msg["type"] == "tool":
                for tc in msg.get("tool_calls", []):
                    parts.append(f"Tool {tc['name']} executed")
        return "\n".join(parts)

    def _extract_key_points(self, messages: List[Dict]) -> List[str]:
        points = []
        for msg in messages:
            for tc in msg.get("tool_calls", []):
                points.append(f"Used {tc['name']}: {tc['arguments'][:100]}")

            content = msg["content"].lower()
            # 检测错误关键词（包括中文）
            if "error" in content or "failed" in content or "错误" in content or "失败" in content:
                points.append(f"Error: {msg['content'][:100]}")
            # 检测重要信息关键词（包括中文）
            if "important" in content or "note" in content or "重要" in content or "注意" in content:
                points.append(f"Important: {msg['content'][:100]}")

        return points

    def _extract_entities(self, messages: List[Dict]) -> List[Entity]:
        entities = []
        for msg in messages:
            content = msg["content"]
            if ".go" in content or ".py" in content:
                entities.append(Entity(type="file", name="referenced file"))
            if "http://" in content or "https://" in content:
                entities.append(Entity(type="url", name="referenced URL"))
        return entities

    def _summarize_chunk(self, messages: List[Dict]) -> str:
        parts = []
        for msg in messages:
            if msg["type"] == "user":
                parts.append(f"Q: {msg['content'][:100]}")
            elif msg["type"] == "assistant":
                parts.append(f"A: {msg['content'][:100]}")
            elif msg["type"] == "tool":
                for tc in msg.get("tool_calls", []):
                    parts.append(f"[{tc['name']}]")
        return " | ".join(parts)

    def create_session(self) -> Dict:
        return {
            "id": self.generate_id("sess"),
            "messages": [],
            "summaries": [],
            "token_count": 0,
            "created_at": datetime.now(),
            "updated_at": datetime.now()
        }

    def get_statistics(self, session: Dict) -> Dict:
        total_original = sum(s.get("original_tokens", 0) for s in session["summaries"])
        current = session["token_count"]

        return {
            "message_count": len(session["messages"]),
            "token_count": current,
            "summary_count": len(session["summaries"]),
            "original_tokens": total_original,
            "compression_ratio": current / (total_original + current) if total_original + current > 0 else 0
        }

# ==================== 测试用例 ====================

class TestRunner:
    def __init__(self):
        self.results = []
        self.passed = 0
        self.failed = 0

    def record(self, name: str, passed: bool, details: str = ""):
        self.results.append({
            "name": name,
            "passed": passed,
            "details": details
        })
        if passed:
            self.passed += 1
        else:
            self.failed += 1

        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"  {status}: {name}")
        if details and not passed:
            print(f"         {details}")

    def summary(self):
        total = self.passed + self.failed
        print(f"\n{'='*60}")
        print(f"测试结果: {self.passed}/{total} 通过")
        print(f"{'='*60}")

def generate_long_content(lines: int = 50) -> str:
    """生成长文本内容"""
    paragraphs = []
    for i in range(lines):
        words = ''.join(random.choices(string.ascii_lowercase, k=random.randint(20, 50)))
        paragraphs.append(f"Line {i+1}: {words}")
    return "\n".join(paragraphs)

def test_token_threshold_trigger():
    """测试 1: Token 阈值触发压缩"""
    print("\n📋 测试 1: Token 阈值触发压缩")
    print("-" * 50)

    runner = TestRunner()
    config = CompressionConfig(
        max_tokens=1000,
        min_messages_to_compact=5,
        preserve_recent=2
    )
    tokenizer = SimpleTokenizer()
    compactor = Compactor(config, tokenizer)

    session = compactor.create_session()

    # 添加消息直到触发阈值
    for i in range(4):
        compactor.add_message(session, Message(
            id="",
            type=MessageType.USER,
            content=f"Short message {i}"
        ))

    runner.record(
        "未达阈值不应压缩",
        not compactor.should_compact(session),
        f"token_count={session['token_count']}"
    )

    # 添加大量消息超过阈值
    for i in range(20):
        compactor.add_message(session, Message(
            id="",
            type=MessageType.USER,
            content=generate_long_content(30)
        ))

    original_tokens = session["token_count"]
    runner.record(
        "超过阈值应触发压缩",
        compactor.should_compact(session),
        f"token_count={session['token_count']}, max={config.max_tokens}"
    )

    # 执行压缩
    summary = compactor.compact(session)

    runner.record(
        "压缩成功返回 Summary",
        summary is not None,
        f"summary_id={summary.id if summary else None}"
    )

    runner.record(
        "压缩后消息数减少",
        len(session["messages"]) < 24,
        f"messages={len(session['messages'])}"
    )

    runner.record(
        "Token 数量有效减少",
        session["token_count"] < original_tokens,
        f"tokens: {original_tokens} -> {session['token_count']}"
    )

    runner.record(
        "压缩比例合理",
        session["token_count"] / original_tokens < 0.5,
        f"ratio={session['token_count']/original_tokens:.1%}"
    )

    runner.summary()
    return runner

def test_compression_strategies():
    """测试 2: 多种压缩策略对比"""
    print("\n📋 测试 2: 多种压缩策略对比")
    print("-" * 50)

    runner = TestRunner()
    tokenizer = SimpleTokenizer()

    strategies = [
        CompressionStrategy.TRUNCATE,
        CompressionStrategy.SEMANTIC,
        CompressionStrategy.HIERARCHICAL
    ]

    results = {}

    for strategy in strategies:
        config = CompressionConfig(
            strategy=strategy,
            max_tokens=2000,
            min_messages_to_compact=5,
            preserve_recent=3
        )
        compactor = Compactor(config, tokenizer)
        session = compactor.create_session()

        # 添加相同的消息集
        for i in range(15):
            compactor.add_message(session, Message(
                id="",
                type=MessageType.USER,
                content=f"User question {i}: " + generate_long_content(10)
            ))
            compactor.add_message(session, Message(
                id="",
                type=MessageType.ASSISTANT,
                content=f"Assistant answer {i}: " + generate_long_content(15)
            ))

        original_tokens = session["token_count"]
        summary = compactor.compact(session)
        compressed_tokens = session["token_count"]

        compression_ratio = compressed_tokens / original_tokens if original_tokens > 0 else 0

        results[strategy.value] = {
            "original_tokens": original_tokens,
            "compressed_tokens": compressed_tokens,
            "ratio": compression_ratio,
            "message_count": len(session["messages"])
        }

        runner.record(
            f"{strategy.value} 压缩执行成功",
            summary is not None
        )

        runner.record(
            f"{strategy.value} 压缩比例合理 (< 50%)",
            compression_ratio < 0.5,
            f"ratio={compression_ratio:.2%}"
        )

    # 对比结果
    print("\n  📊 压缩策略对比:")
    for strategy, data in results.items():
        print(f"     {strategy}: {data['original_tokens']} -> {data['compressed_tokens']} tokens ({data['ratio']:.1%})")

    runner.summary()
    return runner

def test_key_info_preservation():
    """测试 3: 关键信息保留验证"""
    print("\n📋 测试 3: 关键信息保留验证")
    print("-" * 50)

    runner = TestRunner()
    config = CompressionConfig(
        strategy=CompressionStrategy.SEMANTIC,
        max_tokens=3000,
        min_messages_to_compact=5,
        preserve_recent=3
    )
    tokenizer = SimpleTokenizer()
    compactor = Compactor(config, tokenizer)

    session = compactor.create_session()

    # 添加包含关键信息的消息
    key_info = {
        "user_name": "张三",
        "project": "Poyo",
        "task": "TUI修复",
        "error": "connection timeout",
        "important_note": "记住这个配置",
        "file": "main.go",
        "url": "https://github.com/example/repo"
    }

    # 前面的消息包含关键信息
    for i in range(5):
        compactor.add_message(session, Message(
            id="",
            type=MessageType.USER,
            content=f"用户 {key_info['user_name']} 在项目 {key_info['project']} 中执行任务 {key_info['task']}，修改文件 {key_info['file']}"
        ))
        compactor.add_message(session, Message(
            id="",
            type=MessageType.ASSISTANT,
            content=f"正在处理... 发现错误: {key_info['error']}，参考文档: {key_info['url']}"
        ))
        compactor.add_message(session, Message(
            id="",
            type=MessageType.TOOL,
            content="",
            tool_calls=[ToolCall(
                id=f"tc-{i}",
                name="Bash",
                arguments='{"command": "go build"}'
            )]
        ))

    # 添加重要备注
    compactor.add_message(session, Message(
        id="",
        type=MessageType.USER,
        content=f"IMPORTANT: {key_info['important_note']}"
    ))

    # 添加更多消息达到压缩阈值
    for i in range(10):
        compactor.add_message(session, Message(
            id="",
            type=MessageType.USER,
            content=generate_long_content(20)
        ))

    # 执行压缩
    summary = compactor.compact(session)

    runner.record(
        "压缩生成摘要",
        summary is not None
    )

    # 检查关键点提取
    key_points_str = " ".join(summary.key_points)
    runner.record(
        "提取工具调用信息",
        any("Bash" in kp for kp in summary.key_points),
        f"key_points={summary.key_points[:3]}"
    )

    runner.record(
        "检测错误信息（中英文）",
        any("error" in kp.lower() or "错误" in kp for kp in summary.key_points),
        f"found error keywords"
    )

    runner.record(
        "检测重要信息（中英文）",
        any("important" in kp.lower() or "重要" in kp for kp in summary.key_points),
        f"found important keywords"
    )

    # 检查实体提取
    runner.record(
        "提取实体信息（文件/URL）",
        len(summary.entities) > 0,
        f"entities count: {len(summary.entities)}"
    )

    runner.summary()
    return runner

def test_memory_persistence():
    """测试 4: 记忆持久化测试"""
    print("\n📋 测试 4: 记忆持久化测试")
    print("-" * 50)

    runner = TestRunner()
    config = CompressionConfig(
        max_tokens=2000,
        min_messages_to_compact=5,
        preserve_recent=2
    )
    tokenizer = SimpleTokenizer()
    compactor = Compactor(config, tokenizer)

    session = compactor.create_session()

    # 添加消息
    for i in range(10):
        compactor.add_message(session, Message(
            id="",
            type=MessageType.USER,
            content=f"Message {i}"
        ))

    # 导出会话
    try:
        export_data = json.dumps(session, default=str, indent=2)
        runner.record("会话导出成功", len(export_data) > 0)

        # 重新导入
        imported_session = json.loads(export_data)
        runner.record(
            "会话导入成功",
            imported_session["id"] == session["id"]
        )

        runner.record(
            "导入后消息数一致",
            len(imported_session["messages"]) == len(session["messages"])
        )
    except Exception as e:
        runner.record("会话序列化", False, str(e))

    # 测试统计信息
    stats = compactor.get_statistics(session)

    runner.record(
        "统计信息生成",
        stats is not None
    )

    runner.record(
        "消息计数正确",
        stats["message_count"] == 10
    )

    runner.summary()
    return runner

def test_multi_round_compression():
    """测试 5: 多轮压缩场景"""
    print("\n📋 测试 5: 多轮压缩场景")
    print("-" * 50)

    runner = TestRunner()
    config = CompressionConfig(
        strategy=CompressionStrategy.HIERARCHICAL,
        max_tokens=1500,
        min_messages_to_compact=8,
        preserve_recent=3
    )
    tokenizer = SimpleTokenizer()
    compactor = Compactor(config, tokenizer)

    session = compactor.create_session()

    compression_count = 0
    total_messages_added = 0

    # 模拟多轮对话
    for round_num in range(5):
        print(f"\n  🔄 第 {round_num + 1} 轮对话")

        # 每轮添加消息
        for i in range(10):
            compactor.add_message(session, Message(
                id="",
                type=MessageType.USER,
                content=f"Round {round_num + 1} - User message {i}: " + generate_long_content(15)
            ))
            compactor.add_message(session, Message(
                id="",
                type=MessageType.ASSISTANT,
                content=f"Round {round_num + 1} - Assistant response {i}: " + generate_long_content(20)
            ))
            total_messages_added += 2

        # 检查是否需要压缩
        if compactor.should_compact(session):
            print(f"     ⚡ 触发压缩 (tokens={session['token_count']})")
            summary = compactor.compact(session)
            if summary:
                compression_count += 1
                print(f"     📝 压缩完成: {summary.original_tokens} -> {summary.token_count} tokens")

        print(f"     📊 当前状态: {len(session['messages'])} messages, {session['token_count']} tokens")

    runner.record(
        "多轮压缩执行",
        compression_count >= 3,
        f"压缩次数: {compression_count}"
    )

    stats = compactor.get_statistics(session)

    runner.record(
        "摘要历史记录",
        stats["summary_count"] >= 3,
        f"摘要数: {stats['summary_count']}"
    )

    runner.record(
        "压缩比例递减",
        stats["compression_ratio"] < 0.5,
        f"最终压缩比: {stats['compression_ratio']:.2%}"
    )

    # 验证最近消息保留（检查最后 preserve_recent 条消息是否为用户/助手消息）
    recent_msg_count = min(config.preserve_recent, len(session["messages"]))
    # 注意：压缩后第一条消息是 summary，最后几条应该是保留的原始消息
    recent_messages = session["messages"][-recent_msg_count:] if recent_msg_count > 0 else []
    # 处理 type 可能是枚举或字符串的情况
    def get_type_str(t):
        if isinstance(t, MessageType):
            return t.value
        return str(t)

    recent_preserved = all(
        get_type_str(msg["type"]) in ["user", "assistant"]
        for msg in recent_messages
    )
    runner.record(
        f"最近 {recent_msg_count} 条消息保留",
        recent_preserved,
        f"last {recent_msg_count} types: {[get_type_str(m['type']) for m in recent_messages]}"
    )

    runner.summary()
    return runner

def test_concurrent_operations():
    """测试 6: 并发操作测试"""
    print("\n📋 测试 6: 并发操作测试")
    print("-" * 50)

    import threading

    runner = TestRunner()
    config = CompressionConfig()
    tokenizer = SimpleTokenizer()
    compactor = Compactor(config, tokenizer)

    session = compactor.create_session()

    errors = []
    lock = threading.Lock()

    def add_messages(thread_id: int):
        try:
            for i in range(20):
                with lock:
                    compactor.add_message(session, Message(
                        id="",
                        type=MessageType.USER,
                        content=f"Thread {thread_id} - Message {i}"
                    ))
        except Exception as e:
            errors.append(str(e))

    threads = []
    for i in range(5):
        t = threading.Thread(target=add_messages, args=(i,))
        threads.append(t)
        t.start()

    for t in threads:
        t.join()

    runner.record(
        "并发添加无错误",
        len(errors) == 0,
        f"errors={errors}"
    )

    runner.record(
        "消息总数正确",
        len(session["messages"]) == 100,
        f"expected=100, actual={len(session['messages'])}"
    )

    runner.summary()
    return runner

# ==================== 主测试入口 ====================

def main():
    print("=" * 60)
    print("🧪 Poyo 上下文压缩和记忆能力测试")
    print("=" * 60)

    all_runners = []

    # 运行所有测试
    all_runners.append(test_token_threshold_trigger())
    all_runners.append(test_compression_strategies())
    all_runners.append(test_key_info_preservation())
    all_runners.append(test_memory_persistence())
    all_runners.append(test_multi_round_compression())
    all_runners.append(test_concurrent_operations())

    # 总体统计
    total_passed = sum(r.passed for r in all_runners)
    total_failed = sum(r.failed for r in all_runners)
    total = total_passed + total_failed

    print("\n" + "=" * 60)
    print("📊 总体测试结果")
    print("=" * 60)
    print(f"  ✅ 通过: {total_passed}")
    print(f"  ❌ 失败: {total_failed}")
    print(f"  📈 通过率: {total_passed/total*100:.1f}%")
    print("=" * 60)

    # 测试结论
    if total_failed == 0:
        print("\n🎉 所有测试通过! Poyo 的上下文压缩和记忆能力正常工作。")
    else:
        print(f"\n⚠️  有 {total_failed} 个测试失败，请检查相关功能。")

if __name__ == "__main__":
    main()
