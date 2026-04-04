#!/usr/bin/env python3
"""
Poyo 交互式多轮对话记忆测试 (修复版)
模拟真实用户场景，测试对话记忆和信息保留能力
"""

import json
import os
import time
import re
from datetime import datetime
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, field, asdict
import threading

# ==================== 数据结构 ====================

@dataclass
class ConversationMessage:
    role: str
    content: str
    timestamp: int
    metadata: Dict = field(default_factory=dict)

@dataclass
class MemoryEntry:
    key: str
    value: str
    namespace: str
    created_at: int
    last_accessed: int
    access_count: int
    source: str

# ==================== Poyo 对话系统 ====================

class PoyoConversationSystem:
    """模拟 Poyo 的完整对话+记忆系统"""

    def __init__(self, data_dir: str):
        self.data_dir = data_dir
        self.conversation_file = os.path.join(data_dir, "conversation.json")
        self.memory_file = os.path.join(data_dir, "memory.json")

        self.conversation_history: List[ConversationMessage] = []
        self.memory_store: Dict[str, Dict[str, MemoryEntry]] = {}
        self._lock = threading.RLock()

        os.makedirs(data_dir, exist_ok=True)
        self._load()

    def _load(self):
        """加载持久化数据"""
        if os.path.exists(self.conversation_file):
            try:
                with open(self.conversation_file, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                    self.conversation_history = [
                        ConversationMessage(**msg) for msg in data.get("messages", [])
                    ]
            except:
                pass

        if os.path.exists(self.memory_file):
            try:
                with open(self.memory_file, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                    for ns, entries in data.items():
                        self.memory_store[ns] = {
                            k: MemoryEntry(**v) for k, v in entries.items()
                        }
            except:
                pass

    def _save(self):
        """持久化数据"""
        with open(self.conversation_file, 'w', encoding='utf-8') as f:
            json.dump({
                "messages": [asdict(msg) for msg in self.conversation_history]
            }, f, ensure_ascii=False, indent=2)

        with open(self.memory_file, 'w', encoding='utf-8') as f:
            data = {}
            for ns, entries in self.memory_store.items():
                data[ns] = {k: asdict(v) for k, v in entries.items()}
            json.dump(data, f, ensure_ascii=False, indent=2)

    def chat(self, user_message: str) -> str:
        """处理用户消息"""
        with self._lock:
            timestamp = int(time.time())

            # 记录用户消息
            self.conversation_history.append(ConversationMessage(
                role="user",
                content=user_message,
                timestamp=timestamp
            ))

            # 提取信息
            extracted_info = self._extract_info(user_message)

            # 存储新信息
            for key, value in extracted_info.items():
                self._remember(key, value, source="conversation")

            # 检索相关记忆
            relevant_memories = self._recall_relevant(user_message)

            # 生成响应
            response = self._generate_response(user_message, extracted_info, relevant_memories)

            # 记录响应
            self.conversation_history.append(ConversationMessage(
                role="assistant",
                content=response,
                timestamp=timestamp,
                metadata={"extracted": extracted_info, "recalled": list(relevant_memories.keys())}
            ))

            self._save()
            return response

    def _extract_info(self, message: str) -> Dict:
        """从消息中提取关键信息"""
        info = {}

        # 提取用户名
        name_patterns = [
            r"我叫([^\s，。！？]+)",
            r"我是([^\s，。！？]+)",
        ]
        for pattern in name_patterns:
            match = re.search(pattern, message)
            if match:
                name = match.group(1).strip()
                # 排除疑问词和常见干扰词
                exclude_words = ["谁", "什么", "哪", "怎么", "为什么", "如何"]
                if len(name) <= 10 and name and name not in exclude_words:
                    info["user_name"] = name
                    break

        # 提取项目名 - 改进模式
        project_patterns = [
            r"做[一]?个[叫是]?\s*([^\s，。！？]+)\s*的?项目",
            r"项目[叫是]\s*([^\s，。！？]+)",
            r"叫([^\s，。！？]+)的?项目",
        ]
        for pattern in project_patterns:
            match = re.search(pattern, message)
            if match:
                project = match.group(1).strip()
                # 排除疑问词
                exclude_words = ["什么", "哪", "谁", "怎么", "为什么", "如何", "的"]
                if project and project not in exclude_words:
                    info["project"] = project
                    break

        # 提取偏好
        if re.search(r"我喜欢|偏好|习惯用", message):
            info["preference"] = message

        # 提取邮箱
        email_match = re.search(r'[\w\.-]+@[\w\.-]+\.\w+', message)
        if email_match:
            info["email"] = email_match.group(0)

        return info

    def _remember(self, key: str, value: str, namespace: str = "default", source: str = "conversation"):
        """存储记忆"""
        timestamp = int(time.time())

        if namespace not in self.memory_store:
            self.memory_store[namespace] = {}

        # 如果已存在，更新访问计数
        if key in self.memory_store[namespace]:
            entry = self.memory_store[namespace][key]
            entry.value = value  # 更新值
            entry.last_accessed = timestamp
            entry.access_count += 1
        else:
            self.memory_store[namespace][key] = MemoryEntry(
                key=key,
                value=value,
                namespace=namespace,
                created_at=timestamp,
                last_accessed=timestamp,
                access_count=1,
                source=source
            )

    def _recall_relevant(self, query: str) -> Dict[str, MemoryEntry]:
        """检索相关记忆"""
        results = {}

        # 关键词映射
        keyword_map = {
            "名字": "user_name",
            "叫什么": "user_name",
            "是谁": "user_name",
            "项目": "project",
            "做什么": "project",
            "偏好": "preference",
            "喜欢": "preference",
            "邮箱": "email",
            "联系": "email",
            "总结": "__all__",  # 特殊标记：返回所有记忆
            "关于我": "__all__",
            "我的信息": "__all__",
        }

        # 检查是否需要返回所有记忆
        for query_keyword, memory_key in keyword_map.items():
            if query_keyword in query and memory_key == "__all__":
                # 返回所有记忆
                for ns, entries in self.memory_store.items():
                    for key, entry in entries.items():
                        results[key] = entry
                        entry.last_accessed = int(time.time())
                        entry.access_count += 1
                return results

        # 普通关键词匹配
        for ns, entries in self.memory_store.items():
            for key, entry in entries.items():
                # 直接键匹配
                for query_keyword, memory_key in keyword_map.items():
                    if query_keyword in query and key == memory_key:
                        results[key] = entry
                        entry.last_accessed = int(time.time())
                        entry.access_count += 1
                        break

                # 值匹配（排除疑问词）
                if key not in results:
                    exclude_keywords = ["什么", "怎么", "为什么", "如何", "哪"]
                    for query_keyword in query.split():
                        if len(query_keyword) >= 2 and query_keyword not in exclude_keywords:
                            if query_keyword in entry.value:
                                results[key] = entry
                                entry.last_accessed = int(time.time())
                                entry.access_count += 1
                                break

        return results

    def _generate_response(self, user_message: str, extracted_info: Dict, relevant_memories: Dict) -> str:
        """生成响应"""
        response_parts = []

        lower_msg = user_message.lower()

        # 处理记忆查询（疑问句优先级最高）
        if "记得" in user_message or "什么" in user_message or "总结" in user_message or "关于" in user_message:
            if relevant_memories:
                if "总结" in user_message or "关于我" in user_message or "我的信息" in user_message:
                    response_parts.append("关于你，我知道以下信息：")
                    for key, entry in relevant_memories.items():
                        response_parts.append(f"  - {key}: {entry.value}")
                else:
                    for key, entry in relevant_memories.items():
                        if key == "user_name":
                            response_parts.append(f"是的，我记得你叫 {entry.value}")
                        elif key == "project":
                            response_parts.append(f"你的项目是 {entry.value}")
                        elif key == "preference":
                            response_parts.append(f"你告诉我 {entry.value}")
                        elif key == "email":
                            response_parts.append(f"你的邮箱是 {entry.value}")
            else:
                if "名字" in user_message:
                    response_parts.append("抱歉，我暂时不记得你的名字。你能再告诉我一次吗？")
                elif "项目" in user_message:
                    response_parts.append("我还没有记录你的项目信息。")
                else:
                    response_parts.append("我暂时没有相关的记忆。")

        # 处理问候
        elif "你好" in user_message or "hello" in lower_msg:
            if "user_name" in relevant_memories:
                response_parts.append(f"你好，{relevant_memories['user_name'].value}！很高兴又见到你！")
            else:
                response_parts.append("你好！我是 Poyo，很高兴认识你！")

        # 处理自我介绍（有新信息提取时）
        elif extracted_info:
            for key, value in extracted_info.items():
                if key == "user_name":
                    response_parts.append(f"好的，{value}！我记住你的名字了。")
                elif key == "project":
                    response_parts.append(f"好的，我记住了你的项目「{value}」。")
                elif key == "email":
                    response_parts.append(f"好的，我记住了你的邮箱：{value}")
                else:
                    response_parts.append(f"好的，我记下了。")

        # 处理状态查询
        elif "状态" in user_message:
            stats = self.get_stats()
            response_parts.append(f"当前对话轮数: {stats['conversation_turns']}")
            response_parts.append(f"记忆条数: {stats['memory_count']}")
            if stats['memory_count'] > 0:
                response_parts.append("记忆内容:")
                for ns, entries in self.memory_store.items():
                    for key, entry in entries.items():
                        response_parts.append(f"  - {key}: {entry.value}")

        else:
            response_parts.append("我明白了。还有什么我可以帮你的吗？")

        return "\n".join(response_parts)

    def recall_memory(self, key: str, namespace: str = "default") -> Optional[MemoryEntry]:
        """主动检索记忆"""
        with self._lock:
            if namespace in self.memory_store:
                entry = self.memory_store[namespace].get(key)
                if entry:
                    entry.last_accessed = int(time.time())
                    entry.access_count += 1
                    self._save()
                return entry
        return None

    def get_stats(self) -> Dict:
        """获取系统状态"""
        return {
            "conversation_turns": len(self.conversation_history) // 2,
            "memory_count": sum(len(e) for e in self.memory_store.values()),
            "namespaces": list(self.memory_store.keys())
        }


# ==================== 交互式测试 ====================

def run_interactive_test():
    """运行交互式多轮对话测试"""
    print("=" * 70)
    print("🧪 Poyo 交互式多轮对话记忆测试")
    print("=" * 70)
    print("\n模拟场景: 用户与 Poyo 进行多轮对话，测试记忆能力")
    print("=" * 70)

    # 清理旧数据，创建新实例
    data_dir = "/tmp/poyo_interactive_test_v2"
    import shutil
    if os.path.exists(data_dir):
        shutil.rmtree(data_dir)

    poyo = PoyoConversationSystem(data_dir)

    # 测试对话流程
    test_flow = [
        {
            "turn": 1,
            "user": "你好，我叫张三",
            "description": "用户自我介绍"
        },
        {
            "turn": 2,
            "user": "我正在做一个叫 Poyo 的项目",
            "description": "用户介绍项目"
        },
        {
            "turn": 3,
            "user": "你还记得我的名字吗？",
            "description": "测试名字记忆"
        },
        {
            "turn": 4,
            "user": "我喜欢用 Go 语言开发",
            "description": "用户表达偏好"
        },
        {
            "turn": 5,
            "user": "我的项目是什么？",
            "description": "测试项目记忆"
        },
        {
            "turn": 6,
            "user": "我的邮箱是 zhangsan@example.com",
            "description": "用户提供联系方式"
        },
        {
            "turn": 7,
            "user": "总结一下你知道关于我的信息",
            "description": "综合记忆测试"
        },
        {
            "turn": 8,
            "user": "当前状态",
            "description": "查看系统状态"
        }
    ]

    print("\n" + "=" * 70)
    print("📝 开始多轮对话测试")
    print("=" * 70)

    results = []

    for turn_data in test_flow:
        turn_num = turn_data["turn"]
        user_input = turn_data["user"]
        description = turn_data["description"]

        print(f"\n{'─' * 60}")
        print(f"🔄 第 {turn_num} 轮对话")
        print(f"{'─' * 60}")
        print(f"📋 测试点: {description}")

        print(f"\n👤 用户输入:")
        print(f"   \"{user_input}\"")

        response = poyo.chat(user_input)

        print(f"\n💚 Poyo 响应:")
        for line in response.split('\n'):
            print(f"   {line}")

        results.append({
            "turn": turn_num,
            "description": description,
            "user": user_input,
            "response": response
        })

    # 最终统计
    print("\n" + "=" * 70)
    print("📊 最终记忆状态")
    print("=" * 70)

    stats = poyo.get_stats()
    print(f"\n对话轮数: {stats['conversation_turns']}")
    print(f"记忆条数: {stats['memory_count']}")

    print("\n记忆内容:")
    for ns, entries in poyo.memory_store.items():
        print(f"\n  [{ns}]")
        for key, entry in entries.items():
            print(f"    {key}: {entry.value}")
            print(f"      创建于: {datetime.fromtimestamp(entry.created_at)}")
            print(f"      访问次数: {entry.access_count}")

    # 验证记忆完整性
    print("\n" + "=" * 70)
    print("🔍 记忆完整性验证")
    print("=" * 70)

    expected_memories = {
        "user_name": "张三",
        "project": "Poyo",
        "preference": "Go",
        "email": "zhangsan@example.com"
    }

    all_correct = True
    for key, expected_value in expected_memories.items():
        entry = poyo.recall_memory(key)
        if entry:
            match = expected_value in entry.value
            status = "✅" if match else "❌"
            print(f"   {status} {key}: '{entry.value}' {'包含' if match else '不包含'} '{expected_value}'")
            if not match:
                all_correct = False
        else:
            print(f"   ❌ {key}: 未找到")
            all_correct = False

    print("\n" + "=" * 70)
    if all_correct:
        print("🎉 所有记忆完整！Poyo 能正确记住对话内容")
    else:
        print("⚠️  部分记忆缺失或错误")
    print("=" * 70)


if __name__ == "__main__":
    run_interactive_test()
