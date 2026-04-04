#!/usr/bin/env python3
"""
Poyo 交互式对话测试
真正的交互式多轮对话，测试记忆能力
"""

import json
import os
import re
import time
from dataclasses import dataclass, field, asdict
from typing import Dict, Optional
from pathlib import Path

# 颜色输出
class Colors:
    GREEN = '\033[92m'
    BLUE = '\033[94m'
    YELLOW = '\033[93m'
    CYAN = '\033[96m'
    RED = '\033[91m'
    BOLD = '\033[1m'
    END = '\033[0m'

def print_header():
    print(f"\n{Colors.CYAN}{'='*60}{Colors.END}")
    print(f"{Colors.CYAN}🤖 Poyo 交互式对话测试{Colors.END}")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}")
    print(f"\n{Colors.YELLOW}提示：{Colors.END}")
    print("  - 直接输入消息与 Poyo 对话")
    print("  - 输入 '状态' 查看记忆状态")
    print("  - 输入 '记忆' 查看所有记忆")
    print("  - 输入 '清空' 清除所有记忆")
    print("  - 输入 '退出' 或 'quit' 结束对话")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}\n")


@dataclass
class MemoryEntry:
    key: str
    value: str
    namespace: str = "default"
    created_at: int = 0
    last_accessed: int = 0
    access_count: int = 0
    metadata: Dict = field(default_factory=dict)

    def __post_init__(self):
        if not self.created_at:
            self.created_at = int(time.time())
        if not self.last_accessed:
            self.last_accessed = self.created_at


class PoyoInteractive:
    def __init__(self, memory_file: str = None):
        self.conversation_turns = 0
        self.memory_store: Dict[str, Dict[str, MemoryEntry]] = {"default": {}}

        if memory_file:
            self.memory_file = Path(memory_file)
            self._load_persistence()
        else:
            self.memory_file = None

    def _load_persistence(self):
        """加载持久化记忆"""
        if self.memory_file and self.memory_file.exists():
            try:
                with open(self.memory_file, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                    for ns, entries in data.get('namespaces', {}).items():
                        self.memory_store[ns] = {}
                        for key, entry in entries.items():
                            self.memory_store[ns][key] = MemoryEntry(**entry)
            except Exception as e:
                print(f"{Colors.RED}加载记忆失败: {e}{Colors.END}")

    def _save_persistence(self):
        """保存记忆"""
        if not self.memory_file:
            return
        try:
            self.memory_file.parent.mkdir(parents=True, exist_ok=True)
            data = {'namespaces': {}}
            for ns, entries in self.memory_store.items():
                data['namespaces'][ns] = {k: asdict(v) for k, v in entries.items()}
            with open(self.memory_file, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
        except Exception as e:
            print(f"{Colors.RED}保存记忆失败: {e}{Colors.END}")

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
                exclude_words = ["谁", "什么", "哪", "怎么", "为什么", "如何"]
                if len(name) <= 10 and name and name not in exclude_words:
                    info["user_name"] = name
                    break

        # 提取项目名
        project_patterns = [
            r"做[一]?个[叫是]?\s*([^\s，。！？]+)\s*的?项目",
            r"项目[叫是]\s*([^\s，。！？]+)",
            r"叫([^\s，。！？]+)的?项目",
        ]
        for pattern in project_patterns:
            match = re.search(pattern, message)
            if match:
                project = match.group(1).strip()
                exclude_words = ["什么", "哪", "谁", "怎么", "为什么", "如何", "的"]
                if project and project not in exclude_words:
                    info["project"] = project
                    break

        # 提取偏好
        if re.search(r"我喜欢|偏好|习惯用|爱用", message):
            info["preference"] = message

        # 提取邮箱
        email_match = re.search(r'[\w\.-]+@[\w\.-]+\.\w+', message)
        if email_match:
            info["email"] = email_match.group(0)

        # 提取电话
        phone_match = re.search(r'1[3-9]\d{9}', message)
        if phone_match:
            info["phone"] = phone_match.group(0)

        # 提取公司/部门
        dept_patterns = [
            r"在([^\s，。！？]+)(?:工作|上班)",
            r"我是([^\s，。！？]+)的",
            r"就职于([^\s，。！？]+)",
        ]
        exclude_words = ["哪", "哪里", "什么", "谁", "怎么", "为什么", "如何", "这", "那"]
        for pattern in dept_patterns:
            match = re.search(pattern, message)
            if match:
                dept = match.group(1).strip()
                if dept and len(dept) >= 2 and dept not in exclude_words:
                    info["department"] = dept
                    break

        return info

    def _recall_relevant(self, query: str) -> Dict[str, MemoryEntry]:
        """检索相关记忆"""
        results = {}

        keyword_map = {
            "名字": "user_name",
            "叫什么": "user_name",
            "是谁": "user_name",
            "项目": "project",
            "做什么": "project",
            "偏好": "preference",
            "喜欢": "preference",
            "邮箱": "email",
            "邮件": "email",
            "联系": "email",
            "电话": "phone",
            "手机": "phone",
            "部门": "department",
            "公司": "department",
            "在哪": "department",
            "工作": "department",
            "总结": "__all__",
            "关于我": "__all__",
            "我的信息": "__all__",
            "你知道什么": "__all__",
            "都知道": "__all__",
        }

        # 检查是否需要返回所有记忆
        for query_keyword, memory_key in keyword_map.items():
            if query_keyword in query and memory_key == "__all__":
                for ns, entries in self.memory_store.items():
                    for key, entry in entries.items():
                        results[key] = entry
                        entry.last_accessed = int(time.time())
                        entry.access_count += 1
                return results

        # 普通关键词匹配
        for ns, entries in self.memory_store.items():
            for key, entry in entries.items():
                for query_keyword, memory_key in keyword_map.items():
                    if query_keyword in query and key == memory_key:
                        results[key] = entry
                        entry.last_accessed = int(time.time())
                        entry.access_count += 1
                        break

        return results

    def store_memory(self, key: str, value: str, namespace: str = "default", metadata: Dict = None):
        """存储记忆"""
        if namespace not in self.memory_store:
            self.memory_store[namespace] = {}

        entry = MemoryEntry(
            key=key,
            value=value,
            namespace=namespace,
            metadata=metadata or {}
        )
        self.memory_store[namespace][key] = entry
        self._save_persistence()
        return {"status": "success", "key": key}

    def chat(self, user_message: str) -> str:
        """处理对话"""
        self.conversation_turns += 1

        # 先提取信息
        extracted_info = self._extract_info(user_message)

        # 存储提取的信息
        for key, value in extracted_info.items():
            self.store_memory(key, value)

        # 检索相关记忆
        relevant_memories = self._recall_relevant(user_message)

        # 生成响应
        response = self._generate_response(user_message, extracted_info, relevant_memories)

        return response

    def _generate_response(self, user_message: str, extracted_info: Dict, relevant_memories: Dict) -> str:
        """生成响应"""
        response_parts = []
        lower_msg = user_message.lower()

        # 处理记忆查询
        query_keywords = ["记得", "什么", "总结", "关于", "知道", "我的", "在哪", "哪里"]
        if any(kw in user_message for kw in query_keywords):
            if relevant_memories:
                if any(kw in user_message for kw in ["总结", "关于我", "我的信息", "你知道什么", "都知道"]):
                    response_parts.append("关于你，我知道以下信息：")
                    for key, entry in relevant_memories.items():
                        response_parts.append(f"  📌 {key}: {entry.value}")
                else:
                    for key, entry in relevant_memories.items():
                        if key == "user_name":
                            response_parts.append(f"是的，我记得你叫 {entry.value} 😊")
                        elif key == "project":
                            response_parts.append(f"你的项目是「{entry.value}」")
                        elif key == "preference":
                            response_parts.append(f"你告诉我：{entry.value}")
                        elif key == "email":
                            response_parts.append(f"你的邮箱是 {entry.value}")
                        elif key == "phone":
                            response_parts.append(f"你的电话是 {entry.value}")
                        elif key == "department":
                            response_parts.append(f"你在 {entry.value} 工作")
                        else:
                            response_parts.append(f"{key}: {entry.value}")
            else:
                if "名字" in user_message:
                    response_parts.append("抱歉，我暂时不记得你的名字。你能再告诉我一次吗？🤔")
                elif "项目" in user_message:
                    response_parts.append("我还没有记录你的项目信息。")
                else:
                    response_parts.append("我暂时没有相关的记忆。")

        # 处理问候
        elif "你好" in user_message or "hello" in lower_msg or "hi" in lower_msg:
            if "user_name" in self.memory_store.get("default", {}):
                name = self.memory_store["default"]["user_name"].value
                response_parts.append(f"你好，{name}！很高兴又见到你！👋")
            else:
                response_parts.append("你好！我是 Poyo，很高兴认识你！👋")

        # 处理自我介绍
        elif extracted_info:
            for key, value in extracted_info.items():
                if key == "user_name":
                    response_parts.append(f"好的，{value}！我记住你的名字了 ✅")
                elif key == "project":
                    response_parts.append(f"好的，我记住了你的项目「{value}」✅")
                elif key == "email":
                    response_parts.append(f"好的，我记住了你的邮箱：{value} ✅")
                elif key == "phone":
                    response_parts.append(f"好的，我记住了你的电话：{value} ✅")
                elif key == "department":
                    response_parts.append(f"好的，我记住了你在 {value} ✅")
                else:
                    response_parts.append(f"好的，我记下了 ✅")

        # 处理感谢
        elif "谢谢" in user_message or "感谢" in user_message:
            response_parts.append("不客气！有什么需要帮忙的随时找我 😊")

        # 处理告别
        elif "再见" in user_message or "bye" in lower_msg:
            response_parts.append("再见！下次聊 👋")

        else:
            response_parts.append("我明白了。还有什么我可以帮你的吗？")

        return "\n".join(response_parts)

    def get_stats(self) -> Dict:
        """获取统计信息"""
        memory_count = sum(len(entries) for entries in self.memory_store.values())
        return {
            "conversation_turns": self.conversation_turns,
            "memory_count": memory_count,
            "namespaces": list(self.memory_store.keys())
        }

    def show_memories(self) -> str:
        """显示所有记忆"""
        lines = [f"\n{Colors.CYAN}📋 当前记忆列表{Colors.END}"]
        lines.append(f"{Colors.CYAN}{'─'*40}{Colors.END}")

        total = 0
        for ns, entries in self.memory_store.items():
            if entries:
                lines.append(f"\n[{ns}]")
                for key, entry in entries.items():
                    total += 1
                    access_info = f"访问 {entry.access_count} 次"
                    lines.append(f"  📌 {key}: {entry.value}")
                    lines.append(f"     └─ {access_info}")

        lines.append(f"\n{Colors.CYAN}{'─'*40}{Colors.END}")
        lines.append(f"总计: {total} 条记忆")
        return "\n".join(lines)

    def clear_memories(self):
        """清空记忆"""
        self.memory_store = {"default": {}}
        self._save_persistence()
        return "所有记忆已清空 🗑️"


def main():
    print_header()

    # 使用临时目录存储记忆
    memory_file = "/tmp/poyo_interactive_memory/.poyo/memory.json"
    poyo = PoyoInteractive(memory_file)

    print(f"{Colors.GREEN}Poyo 已就绪，开始对话吧！{Colors.END}\n")

    while True:
        try:
            # 读取用户输入
            user_input = input(f"{Colors.BLUE}👤 你: {Colors.END}").strip()

            if not user_input:
                continue

            # 处理命令
            if user_input.lower() in ['退出', 'quit', 'exit', 'q']:
                stats = poyo.get_stats()
                print(f"\n{Colors.YELLOW}📊 对话统计:{Colors.END}")
                print(f"  对话轮数: {stats['conversation_turns']}")
                print(f"  记忆条数: {stats['memory_count']}")
                print(f"\n{Colors.GREEN}再见！感谢使用 Poyo 👋{Colors.END}\n")
                break

            elif user_input in ['状态', 'status']:
                stats = poyo.get_stats()
                print(f"{Colors.YELLOW}📊 当前状态:{Colors.END}")
                print(f"  对话轮数: {stats['conversation_turns']}")
                print(f"  记忆条数: {stats['memory_count']}")
                print()
                continue

            elif user_input in ['记忆', '记忆列表', 'memories']:
                print(poyo.show_memories())
                print()
                continue

            elif user_input in ['清空', '清除', 'clear']:
                print(f"{Colors.RED}{poyo.clear_memories()}{Colors.END}")
                print()
                continue

            elif user_input in ['帮助', 'help']:
                print(f"{Colors.YELLOW}可用命令:{Colors.END}")
                print("  状态 - 查看对话状态")
                print("  记忆 - 查看所有记忆")
                print("  清空 - 清除所有记忆")
                print("  退出 - 结束对话")
                print()
                continue

            # 正常对话
            response = poyo.chat(user_input)
            print(f"{Colors.GREEN}🤖 Poyo: {Colors.END}{response}\n")

        except KeyboardInterrupt:
            print(f"\n\n{Colors.YELLOW}对话中断。再见！{Colors.END}\n")
            break
        except Exception as e:
            print(f"{Colors.RED}错误: {e}{Colors.END}\n")


if __name__ == "__main__":
    main()
