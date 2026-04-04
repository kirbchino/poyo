#!/usr/bin/env python3
"""
Poyo 长期记忆端到端测试
真实测试：文件持久化 → 会话重启 → 数据恢复 → 跨会话验证
"""

import json
import os
import time
import shutil
from datetime import datetime
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, field, asdict
import threading
import hashlib

# ==================== 配置 ====================

TEST_DIR = "/tmp/poyo_e2e_memory_test"
MEMORY_FILE = os.path.join(TEST_DIR, ".poyo", "memory.json")
SESSION_FILE = os.path.join(TEST_DIR, ".poyo", "session.json")

# ==================== 核心类 ====================

@dataclass
class MemoryEntry:
    value: str
    timestamp: int
    access_count: int = 0
    metadata: Dict[str, Any] = field(default_factory=dict)

@dataclass
class SessionState:
    session_id: str
    created_at: int
    messages: List[Dict]
    memory_entries: int
    checksum: str

class PersistentMemoryStore:
    """真正的持久化记忆存储"""

    def __init__(self, memory_file: str):
        self.memory_file = memory_file
        self.memory_store: Dict[str, Dict[str, MemoryEntry]] = {}
        self._lock = threading.RLock()
        self._ensure_dir()

    def _ensure_dir(self):
        os.makedirs(os.path.dirname(self.memory_file), exist_ok=True)

    def store(self, key: str, value: str, namespace: str = "default", metadata: Dict = None) -> Dict:
        """存储并立即持久化"""
        with self._lock:
            if namespace not in self.memory_store:
                self.memory_store[namespace] = {}

            entry = MemoryEntry(
                value=value,
                timestamp=int(time.time()),
                access_count=0,
                metadata=metadata or {}
            )

            self.memory_store[namespace][key] = entry
            self._persist()

            return {
                "success": True,
                "key": key,
                "namespace": namespace,
                "size": len(value),
                "timestamp": entry.timestamp
            }

    def retrieve(self, key: str, namespace: str = "default") -> Dict:
        """检索并更新访问计数"""
        with self._lock:
            if namespace not in self.memory_store:
                return {"success": False, "error": f"namespace not found: {namespace}"}

            entry = self.memory_store[namespace].get(key)
            if not entry:
                return {"success": False, "error": f"key not found: {key}"}

            entry.access_count += 1
            self._persist()  # 持久化访问计数更新

            return {
                "success": True,
                "key": key,
                "value": entry.value,
                "namespace": namespace,
                "timestamp": entry.timestamp,
                "access_count": entry.access_count,
                "metadata": entry.metadata
            }

    def search(self, query: str) -> Dict:
        """搜索记忆"""
        results = []
        query_lower = query.lower()

        with self._lock:
            for ns, entries in self.memory_store.items():
                for key, entry in entries.items():
                    if query_lower in key.lower() or query_lower in entry.value.lower():
                        results.append({
                            "namespace": ns,
                            "key": key,
                            "value": entry.value[:100] + "..." if len(entry.value) > 100 else entry.value,
                            "access_count": entry.access_count,
                            "match_in": "key" if query_lower in key.lower() else "value"
                        })

        return {"success": True, "query": query, "results": results, "total": len(results)}

    def delete(self, key: str, namespace: str = "default") -> Dict:
        """删除并持久化"""
        with self._lock:
            if namespace not in self.memory_store:
                return {"success": False, "error": f"namespace not found: {namespace}"}

            if key not in self.memory_store[namespace]:
                return {"success": False, "error": f"key not found: {key}"}

            del self.memory_store[namespace][key]

            if not self.memory_store[namespace]:
                del self.memory_store[namespace]

            self._persist()

            return {"success": True, "key": key, "namespace": namespace}

    def get_stats(self) -> Dict:
        """获取统计信息"""
        with self._lock:
            namespaces = len(self.memory_store)
            total_keys = sum(len(entries) for entries in self.memory_store.values())
            total_size = sum(
                len(k) + len(e.value)
                for entries in self.memory_store.values()
                for k, e in entries.items()
            )

            return {
                "success": True,
                "stats": {
                    "namespaces": namespaces,
                    "total_keys": total_keys,
                    "total_size": total_size,
                    "memory_file": self.memory_file,
                    "file_exists": os.path.exists(self.memory_file)
                }
            }

    def _persist(self):
        """持久化到文件"""
        data = {}
        for ns, entries in self.memory_store.items():
            data[ns] = {}
            for key, entry in entries.items():
                data[ns][key] = asdict(entry)

        with open(self.memory_file, 'w', encoding='utf-8') as f:
            json.dump(data, f, ensure_ascii=False, indent=2)

    def load(self) -> bool:
        """从文件加载"""
        if not os.path.exists(self.memory_file):
            return False

        try:
            with open(self.memory_file, 'r', encoding='utf-8') as f:
                data = json.load(f)

            self.memory_store = {}
            for ns, entries in data.items():
                self.memory_store[ns] = {}
                for key, entry_data in entries.items():
                    self.memory_store[ns][key] = MemoryEntry(
                        value=entry_data["value"],
                        timestamp=entry_data["timestamp"],
                        access_count=entry_data.get("access_count", 0),
                        metadata=entry_data.get("metadata", {})
                    )
            return True
        except Exception as e:
            print(f"Load error: {e}")
            return False

    def get_checksum(self) -> str:
        """获取存储内容的校验和"""
        content = json.dumps(self.memory_store, sort_keys=True, default=str)
        return hashlib.md5(content.encode()).hexdigest()[:8]

    def export_session_state(self, session_id: str) -> SessionState:
        """导出会话状态"""
        return SessionState(
            session_id=session_id,
            created_at=int(time.time()),
            messages=[],
            memory_entries=sum(len(e) for e in self.memory_store.values()),
            checksum=self.get_checksum()
        )


# ==================== 端到端测试 ====================

def setup_test_environment():
    """设置测试环境"""
    print("=" * 70)
    print("🔧 设置端到端测试环境")
    print("=" * 70)

    # 清理旧测试目录
    if os.path.exists(TEST_DIR):
        shutil.rmtree(TEST_DIR)

    os.makedirs(TEST_DIR, exist_ok=True)
    os.makedirs(os.path.join(TEST_DIR, ".poyo"), exist_ok=True)

    print(f"\n✅ 测试目录: {TEST_DIR}")
    print(f"✅ 记忆文件: {MEMORY_FILE}")

    return True


def test_e2e_session_lifecycle():
    """
    端到端测试 1: 完整会话生命周期
    会话1: 创建记忆 → 持久化 → 关闭
    会话2: 加载记忆 → 验证数据 → 新增记忆
    会话3: 加载记忆 → 验证所有数据
    """
    print("\n" + "=" * 70)
    print("📋 端到端测试 1: 完整会话生命周期")
    print("=" * 70)

    # ========== 会话 1 ==========
    print("\n" + "-" * 50)
    print("📂 会话 1: 创建并持久化记忆")
    print("-" * 50)

    store1 = PersistentMemoryStore(MEMORY_FILE)

    print("\n📥 输入: 存储用户偏好")
    result = store1.store("user_name", "张三", metadata={"source": "user_input"})
    print(f"   key: user_name, value: 张三")
    print(f"📤 输出: success={result['success']}, timestamp={result['timestamp']}")

    result = store1.store("preferred_language", "zh-CN", namespace="preferences")
    print(f"\n📥 输入: 存储偏好设置")
    print(f"   [preferences] preferred_language = zh-CN")
    print(f"📤 输出: success={result['success']}")

    result = store1.store("api_endpoint", "https://api.example.com/v1", namespace="config")
    print(f"\n📥 输入: 存储配置")
    print(f"   [config] api_endpoint = https://api.example.com/v1")
    print(f"📤 输出: success={result['success']}")

    # 获取会话状态
    session1_state = store1.export_session_state("session-1")
    print(f"\n📊 会话 1 状态:")
    print(f"   记忆条目数: {session1_state.memory_entries}")
    print(f"   校验和: {session1_state.checksum}")

    # 验证文件已创建
    print(f"\n✅ 持久化验证:")
    print(f"   文件存在: {os.path.exists(MEMORY_FILE)}")
    print(f"   文件大小: {os.path.getsize(MEMORY_FILE)} bytes")

    # 显示文件内容
    with open(MEMORY_FILE, 'r') as f:
        content = f.read()
    print(f"\n📄 持久化文件内容预览:")
    for line in content.split('\n')[:20]:
        print(f"   {line}")
    print("   ...")

    # 模拟会话结束 - 销毁对象
    del store1

    # ========== 会话 2 ==========
    print("\n" + "-" * 50)
    print("📂 会话 2: 从持久化恢复并新增记忆")
    print("-" * 50)

    store2 = PersistentMemoryStore(MEMORY_FILE)

    print("\n📥 输入: 加载持久化数据")
    loaded = store2.load()
    print(f"📤 输出: 加载成功={loaded}")

    # 验证恢复的数据
    print("\n📥 输入: 检索之前存储的记忆")
    result = store2.retrieve("user_name")
    print(f"   key: user_name")
    print(f"📤 输出: value={result.get('value')}, success={result['success']}")

    result = store2.retrieve("preferred_language", namespace="preferences")
    print(f"\n📥 输入: 检索偏好设置")
    print(f"   [preferences] preferred_language")
    print(f"📤 输出: value={result.get('value')}, success={result['success']}")

    # 新增记忆
    print("\n📥 输入: 新增记忆")
    result = store2.store("last_project", "Poyo", metadata={"session": "session-2"})
    print(f"   last_project = Poyo")
    print(f"📤 输出: success={result['success']}")

    result = store2.store("theme", "dark", namespace="preferences")
    print(f"\n📥 输入: 新增偏好")
    print(f"   [preferences] theme = dark")
    print(f"📤 输出: success={result['success']}")

    session2_state = store2.export_session_state("session-2")
    print(f"\n📊 会话 2 状态:")
    print(f"   记忆条目数: {session2_state.memory_entries}")
    print(f"   校验和: {session2_state.checksum}")

    del store2

    # ========== 会话 3 ==========
    print("\n" + "-" * 50)
    print("📂 会话 3: 最终验证所有数据")
    print("-" * 50)

    store3 = PersistentMemoryStore(MEMORY_FILE)
    store3.load()

    print("\n📥 输入: 列出所有记忆")
    stats = store3.get_stats()["stats"]
    print(f"📤 输出:")
    print(f"   命名空间数: {stats['namespaces']}")
    print(f"   总键数: {stats['total_keys']}")
    print(f"   总大小: {stats['total_size']} bytes")

    # 验证所有记忆
    print("\n📥 输入: 验证所有记忆完整性")
    expected_memories = [
        ("default", "user_name", "张三"),
        ("preferences", "preferred_language", "zh-CN"),
        ("config", "api_endpoint", "https://api.example.com/v1"),
        ("default", "last_project", "Poyo"),
        ("preferences", "theme", "dark"),
    ]

    all_correct = True
    for ns, key, expected_value in expected_memories:
        result = store3.retrieve(key, namespace=ns)
        actual = result.get("value", "")
        match = actual == expected_value
        status = "✅" if match else "❌"
        print(f"   {status} [{ns}] {key}: '{actual}' {'==' if match else '!='} '{expected_value}'")
        if not match:
            all_correct = False

    session3_state = store3.export_session_state("session-3")

    print(f"\n📊 最终状态:")
    print(f"   记忆条目数: {session3_state.memory_entries}")
    print(f"   校验和变化: {session1_state.checksum} → {session2_state.checksum} → {session3_state.checksum}")

    print(f"\n{'✅' if all_correct else '❌'} 结果: {'所有记忆完整保留' if all_correct else '记忆丢失或损坏'}")

    return all_correct


def test_e2e_access_tracking():
    """
    端到端测试 2: 访问计数跨会话追踪
    """
    print("\n" + "=" * 70)
    print("📋 端到端测试 2: 访问计数跨会话追踪")
    print("=" * 70)

    # 会话 1: 创建记忆
    print("\n" + "-" * 50)
    print("📂 会话 1: 创建记忆")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)
    store.store("counter_test", "test_value")
    print("\n📥 输入: 创建记忆 counter_test")
    print("   初始 access_count = 0")

    # 访问 3 次
    for i in range(3):
        result = store.retrieve("counter_test")
        print(f"📤 输出: 第 {i+1} 次访问后 access_count = {result['access_count']}")

    del store

    # 会话 2: 继续访问
    print("\n" + "-" * 50)
    print("📂 会话 2: 继续访问追踪")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)
    store.load()

    result = store.retrieve("counter_test")
    print(f"\n📥 输入: 加载后首次检索")
    print(f"📤 输出: access_count = {result['access_count']} (应为 4)")

    # 再访问 5 次
    for i in range(5):
        result = store.retrieve("counter_test")

    print(f"\n📥 输入: 再访问 5 次")
    print(f"📤 输出: access_count = {result['access_count']} (应为 9)")

    correct = result['access_count'] == 9
    print(f"\n{'✅' if correct else '❌'} 结果: {'访问计数正确追踪' if correct else '访问计数丢失'}")

    del store
    return correct


def test_e2e_search_persistence():
    """
    端到端测试 3: 搜索功能持久化验证
    """
    print("\n" + "=" * 70)
    print("📋 端到端测试 3: 搜索功能持久化验证")
    print("=" * 70)

    # 会话 1: 创建可搜索的记忆
    print("\n" + "-" * 50)
    print("📂 会话 1: 创建记忆")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)

    memories = [
        ("api_key_prod", "sk-prod-12345", "production"),
        ("api_key_dev", "sk-dev-67890", "development"),
        ("db_url_prod", "postgresql://prod.example.com:5432", "production"),
        ("db_url_dev", "postgresql://dev.example.com:5432", "development"),
        ("feature_flag_new_ui", "enabled", "features"),
    ]

    print("\n📥 输入: 存储多条记忆")
    for key, value, ns in memories:
        store.store(key, value, namespace=ns)
        print(f"   [{ns}] {key} = {value[:30]}...")

    # 搜索测试
    print("\n📥 输入: 搜索 'api'")
    result = store.search("api")
    print(f"📤 输出: 找到 {result['total']} 条结果")
    for r in result['results']:
        print(f"   - [{r['namespace']}] {r['key']}")

    del store

    # 会话 2: 搜索持久化的记忆
    print("\n" + "-" * 50)
    print("📂 会话 2: 搜索持久化记忆")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)
    store.load()

    searches = ["api", "prod", "dev", "url", "feature"]

    all_correct = True
    for query in searches:
        print(f"\n📥 输入: 搜索 '{query}'")
        result = store.search(query)
        print(f"📤 输出: 找到 {result['total']} 条结果")
        for r in result['results']:
            print(f"   - [{r['namespace']}] {r['key']} (match in {r['match_in']})")

        if result['total'] == 0 and query in ["api", "prod"]:
            all_correct = False

    print(f"\n{'✅' if all_correct else '❌'} 结果: {'搜索功能正常' if all_correct else '搜索异常'}")

    del store
    return all_correct


def test_e2e_delete_and_recovery():
    """
    端到端测试 4: 删除操作持久化验证
    """
    print("\n" + "=" * 70)
    print("📋 端到端测试 4: 删除操作持久化验证")
    print("=" * 70)

    # 会话 1: 创建记忆
    print("\n" + "-" * 50)
    print("📂 会话 1: 创建记忆")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)

    for i in range(5):
        store.store(f"item_{i}", f"value_{i}")

    stats = store.get_stats()["stats"]
    print(f"\n📥 输入: 创建 5 条记忆")
    print(f"📤 输出: total_keys = {stats['total_keys']}")

    # 删除一条
    print("\n📥 输入: 删除 item_2")
    result = store.delete("item_2")
    print(f"📤 输出: success = {result['success']}")

    del store

    # 会话 2: 验证删除结果
    print("\n" + "-" * 50)
    print("📂 会话 2: 验证删除结果")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)
    store.load()

    stats = store.get_stats()["stats"]
    print(f"\n📥 输入: 加载后检查")
    print(f"📤 输出: total_keys = {stats['total_keys']} (应为 4)")

    # 检索已删除的项
    print("\n📥 输入: 检索已删除的 item_2")
    result = store.retrieve("item_2")
    print(f"📤 输出: success = {result['success']}, error = {result.get('error')}")

    # 检索存在的项
    print("\n📥 输入: 检索存在的 item_3")
    result = store.retrieve("item_3")
    print(f"📤 输出: success = {result['success']}, value = {result.get('value')}")

    correct = stats['total_keys'] == 4 and not store.retrieve("item_2")["success"]
    print(f"\n{'✅' if correct else '❌'} 结果: {'删除操作正确持久化' if correct else '删除状态丢失'}")

    del store
    return correct


def test_e2e_namespace_isolation():
    """
    端到端测试 5: 命名空间隔离持久化验证
    """
    print("\n" + "=" * 70)
    print("📋 端到端测试 5: 命名空间隔离持久化验证")
    print("=" * 70)

    # 会话 1: 在不同命名空间存储同名键
    print("\n" + "-" * 50)
    print("📂 会话 1: 命名空间隔离存储")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)

    namespaces_data = [
        ("production", "config", "prod-config-value"),
        ("staging", "config", "staging-config-value"),
        ("development", "config", "dev-config-value"),
        ("default", "config", "default-config-value"),
    ]

    print("\n📥 输入: 在不同命名空间存储同名键 'config'")
    for ns, key, value in namespaces_data:
        result = store.store(key, value, namespace=ns)
        print(f"   [{ns}] config = {value}")

    del store

    # 会话 2: 验证隔离
    print("\n" + "-" * 50)
    print("📂 会话 2: 验证命名空间隔离")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)
    store.load()

    print("\n📥 输入: 检索各命名空间的 'config'")
    all_correct = True
    for ns, key, expected_value in namespaces_data:
        result = store.retrieve(key, namespace=ns)
        actual = result.get("value", "")
        match = actual == expected_value
        status = "✅" if match else "❌"
        print(f"   {status} [{ns}] config = '{actual}' {'==' if match else '!='} '{expected_value}'")
        if not match:
            all_correct = False

    print(f"\n{'✅' if all_correct else '❌'} 结果: {'命名空间隔离正确' if all_correct else '命名空间隔离失败'}")

    del store
    return all_correct


def test_e2e_metadata_persistence():
    """
    端到端测试 6: 元数据持久化验证
    """
    print("\n" + "=" * 70)
    print("📋 端到端测试 6: 元数据持久化验证")
    print("=" * 70)

    # 会话 1: 创建带元数据的记忆
    print("\n" + "-" * 50)
    print("📂 会话 1: 创建带元数据的记忆")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)

    print("\n📥 输入: 存储带复杂元数据的记忆")
    metadata = {
        "source": "user_input",
        "priority": "high",
        "tags": ["important", "project"],
        "created_by": "张三",
        "related_keys": ["key1", "key2"]
    }
    result = store.store("complex_entry", "important_value", metadata=metadata)
    print(f"   key: complex_entry")
    print(f"   metadata: {json.dumps(metadata, ensure_ascii=False)}")

    del store

    # 会话 2: 验证元数据
    print("\n" + "-" * 50)
    print("📂 会话 2: 验证元数据完整性")
    print("-" * 50)

    store = PersistentMemoryStore(MEMORY_FILE)
    store.load()

    print("\n📥 输入: 检索并检查元数据")
    result = store.retrieve("complex_entry")
    print(f"📤 输出:")
    print(f"   value: {result.get('value')}")
    print(f"   metadata: {json.dumps(result.get('metadata', {}), ensure_ascii=False)}")

    stored_metadata = result.get("metadata", {})
    correct = (
        stored_metadata.get("source") == "user_input" and
        stored_metadata.get("priority") == "high" and
        "important" in stored_metadata.get("tags", [])
    )

    print(f"\n{'✅' if correct else '❌'} 结果: {'元数据完整保留' if correct else '元数据丢失'}")

    del store
    return correct


def test_e2e_concurrent_persistence():
    """
    端到端测试 7: 并发写入持久化验证
    """
    print("\n" + "=" * 70)
    print("📋 端到端测试 7: 并发写入持久化验证")
    print("=" * 70)

    store = PersistentMemoryStore(MEMORY_FILE)
    errors = []
    success_count = [0]
    lock = threading.Lock()

    def writer(thread_id: int, count: int):
        try:
            for i in range(count):
                result = store.store(
                    f"thread_{thread_id}_key_{i}",
                    f"value_{thread_id}_{i}",
                    namespace=f"thread_{thread_id}"
                )
                if result["success"]:
                    with lock:
                        success_count[0] += 1
        except Exception as e:
            errors.append(str(e))

    print("\n📥 输入: 10 个线程并发写入")
    print("   每线程写入 10 条记忆 (共 100 条)")

    threads = [
        threading.Thread(target=writer, args=(i, 10))
        for i in range(10)
    ]

    for t in threads:
        t.start()
    for t in threads:
        t.join()

    print("\n📤 输出:")
    print(f"   错误数: {len(errors)}")
    print(f"   成功数: {success_count[0]}/100")

    # 验证持久化
    stats = store.get_stats()["stats"]
    print(f"   持久化键数: {stats['total_keys']}")

    # 创建新实例验证
    print("\n📥 输入: 创建新实例加载")
    store2 = PersistentMemoryStore(MEMORY_FILE)
    store2.load()
    stats2 = store2.get_stats()["stats"]
    print(f"📤 输出: 加载后键数 = {stats2['total_keys']}")

    correct = (
        len(errors) == 0 and
        success_count[0] == 100 and
        stats2['total_keys'] == 100
    )

    print(f"\n{'✅' if correct else '❌'} 结果: {'并发写入正确持久化' if correct else '并发写入数据丢失'}")

    del store
    del store2
    return correct


# ==================== 主函数 ====================

def main():
    print("=" * 70)
    print("🧪 Poyo 长期记忆端到端测试")
    print("真实测试: 文件持久化 → 会话重启 → 数据恢复 → 跨会话验证")
    print("=" * 70)

    # 设置环境
    setup_test_environment()

    # 运行测试
    tests = [
        ("完整会话生命周期", test_e2e_session_lifecycle),
        ("访问计数跨会话追踪", test_e2e_access_tracking),
        ("搜索功能持久化", test_e2e_search_persistence),
        ("删除操作持久化", test_e2e_delete_and_recovery),
        ("命名空间隔离持久化", test_e2e_namespace_isolation),
        ("元数据持久化", test_e2e_metadata_persistence),
        ("并发写入持久化", test_e2e_concurrent_persistence),
    ]

    results = []
    for name, func in tests:
        try:
            passed = func()
            results.append((name, passed, None))
        except Exception as e:
            results.append((name, False, str(e)))

    # 汇总
    print("\n" + "=" * 70)
    print("📊 端到端测试结果汇总")
    print("=" * 70)

    passed = sum(1 for _, p, _ in results if p)
    total = len(results)

    for name, p, err in results:
        status = "✅ PASS" if p else "❌ FAIL"
        print(f"   {status}: {name}")
        if err:
            print(f"         错误: {err}")

    print(f"\n   总计: {passed}/{total} 通过")

    # 显示最终持久化文件状态
    print("\n" + "-" * 50)
    print("📁 最终持久化文件状态")
    print("-" * 50)

    if os.path.exists(MEMORY_FILE):
        size = os.path.getsize(MEMORY_FILE)
        print(f"   文件路径: {MEMORY_FILE}")
        print(f"   文件大小: {size} bytes")
        print(f"   最后修改: {datetime.fromtimestamp(os.path.getmtime(MEMORY_FILE))}")

    print("\n" + "=" * 70)
    if passed == total:
        print("🎉 所有端到端测试通过! 长期记忆能力正常工作。")
    else:
        print(f"⚠️  有 {total - passed} 个测试失败，请检查持久化逻辑。")
    print("=" * 70)


if __name__ == "__main__":
    main()
