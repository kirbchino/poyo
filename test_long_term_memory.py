#!/usr/bin/env python3
"""
Poyo 长期记忆能力测试
测试场景：
1. 记忆存储与检索
2. 命名空间隔离
3. 持久化与恢复
4. 记忆统计与限制
5. 并发访问安全
6. 记忆生命周期管理
"""

import json
import os
import time
import threading
from datetime import datetime
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, field, asdict
import hashlib

# ==================== 模拟 Poyo Memory Plugin 核心逻辑 ====================

@dataclass
class MemoryEntry:
    """记忆条目"""
    value: str
    timestamp: int
    access_count: int = 0

@dataclass
class MemoryConfig:
    """记忆配置"""
    max_memory_size: int = 10000
    default_namespace: str = "default"
    persistence_file: str = ".poyo/memory.json"

class MemoryStore:
    """模拟 Poyo 的 Memory Plugin"""

    def __init__(self, config: MemoryConfig = None):
        self.config = config or MemoryConfig()
        self.memory_store: Dict[str, Dict[str, MemoryEntry]] = {}
        self._lock = threading.RLock()

    def store(self, key: str, value: str, namespace: str = None) -> Dict:
        """存储记忆"""
        namespace = namespace or self.config.default_namespace

        if not key or not value:
            return {"success": False, "error": "key and value are required"}

        with self._lock:
            # 检查大小限制
            total_size = self._calculate_total_size()
            if total_size + len(key) + len(value) > self.config.max_memory_size:
                return {"success": False, "error": "memory size limit exceeded"}

            # 初始化命名空间
            if namespace not in self.memory_store:
                self.memory_store[namespace] = {}

            # 存储记忆
            self.memory_store[namespace][key] = MemoryEntry(
                value=value,
                timestamp=int(time.time()),
                access_count=0
            )

            # 持久化
            self.save_to_file()

            return {
                "success": True,
                "key": key,
                "namespace": namespace,
                "size": len(value)
            }

    def retrieve(self, key: str, namespace: str = None) -> Dict:
        """检索记忆"""
        namespace = namespace or self.config.default_namespace

        if not key:
            return {"success": False, "error": "key is required"}

        with self._lock:
            if namespace not in self.memory_store:
                return {"success": False, "error": f"namespace not found: {namespace}"}

            entry = self.memory_store[namespace].get(key)
            if not entry:
                return {"success": False, "error": f"memory not found: {key}"}

            # 更新访问计数
            entry.access_count += 1

            return {
                "success": True,
                "key": key,
                "value": entry.value,
                "namespace": namespace,
                "timestamp": entry.timestamp,
                "access_count": entry.access_count
            }

    def list(self, namespace: str = None) -> Dict:
        """列出记忆"""
        result = []

        with self._lock:
            if namespace:
                # 列出特定命名空间
                if namespace in self.memory_store:
                    for key, entry in self.memory_store[namespace].items():
                        result.append({
                            "key": key,
                            "namespace": namespace,
                            "size": len(entry.value),
                            "timestamp": entry.timestamp,
                            "access_count": entry.access_count
                        })
            else:
                # 列出所有
                for ns, entries in self.memory_store.items():
                    for key, entry in entries.items():
                        result.append({
                            "key": key,
                            "namespace": ns,
                            "size": len(entry.value),
                            "timestamp": entry.timestamp,
                            "access_count": entry.access_count
                        })

        # 排序：按时间戳降序
        result.sort(key=lambda x: x["timestamp"], reverse=True)

        return {
            "success": True,
            "memories": result,
            "total": len(result)
        }

    def forget(self, key: str, namespace: str = None) -> Dict:
        """删除记忆"""
        namespace = namespace or self.config.default_namespace

        if not key:
            return {"success": False, "error": "key is required"}

        with self._lock:
            if namespace not in self.memory_store:
                return {"success": False, "error": f"namespace not found: {namespace}"}

            if key not in self.memory_store[namespace]:
                return {"success": False, "error": f"memory not found: {key}"}

            del self.memory_store[namespace][key]

            # 清理空命名空间
            if not self.memory_store[namespace]:
                del self.memory_store[namespace]

            # 持久化
            self.save_to_file()

            return {
                "success": True,
                "key": key,
                "namespace": namespace
            }

    def search(self, query: str, namespace: str = None) -> Dict:
        """搜索记忆（关键词匹配）"""
        results = []
        query_lower = query.lower()

        with self._lock:
            namespaces = [namespace] if namespace else list(self.memory_store.keys())

            for ns in namespaces:
                if ns not in self.memory_store:
                    continue

                for key, entry in self.memory_store[ns].items():
                    # 匹配键或值
                    if query_lower in key.lower() or query_lower in entry.value.lower():
                        results.append({
                            "key": key,
                            "namespace": ns,
                            "value": entry.value,
                            "timestamp": entry.timestamp,
                            "access_count": entry.access_count,
                            "match_type": "key" if query_lower in key.lower() else "value"
                        })

        return {
            "success": True,
            "query": query,
            "results": results,
            "total": len(results)
        }

    def get_stats(self) -> Dict:
        """获取统计信息"""
        with self._lock:
            namespaces = len(self.memory_store)
            total_keys = 0
            total_size = 0

            for ns, entries in self.memory_store.items():
                total_keys += len(entries)
                for key, entry in entries.items():
                    total_size += len(key) + len(entry.value)

            return {
                "success": True,
                "stats": {
                    "namespaces": namespaces,
                    "total_keys": total_keys,
                    "total_size": total_size,
                    "max_size": self.config.max_memory_size,
                    "usage_percent": round(total_size / self.config.max_memory_size * 100, 2)
                }
            }

    def _calculate_total_size(self) -> int:
        """计算总大小"""
        total = 0
        for ns, entries in self.memory_store.items():
            for key, entry in entries.items():
                total += len(key) + len(entry.value)
        return total

    def save_to_file(self) -> None:
        """持久化到文件"""
        try:
            # 确保目录存在
            os.makedirs(os.path.dirname(self.config.persistence_file), exist_ok=True)

            # 序列化
            data = {}
            for ns, entries in self.memory_store.items():
                data[ns] = {}
                for key, entry in entries.items():
                    data[ns][key] = asdict(entry)

            with open(self.config.persistence_file, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
        except Exception as e:
            print(f"Warning: Failed to save memory: {e}")

    def load_from_file(self) -> None:
        """从文件恢复"""
        try:
            if os.path.exists(self.config.persistence_file):
                with open(self.config.persistence_file, 'r', encoding='utf-8') as f:
                    data = json.load(f)

                self.memory_store = {}
                for ns, entries in data.items():
                    self.memory_store[ns] = {}
                    for key, entry_data in entries.items():
                        self.memory_store[ns][key] = MemoryEntry(
                            value=entry_data["value"],
                            timestamp=entry_data["timestamp"],
                            access_count=entry_data.get("access_count", 0)
                        )
        except Exception as e:
            print(f"Warning: Failed to load memory: {e}")

    def clear(self, namespace: str = None) -> Dict:
        """清空记忆"""
        with self._lock:
            if namespace:
                if namespace in self.memory_store:
                    count = len(self.memory_store[namespace])
                    del self.memory_store[namespace]
                    self.save_to_file()
                    return {"success": True, "cleared": count, "namespace": namespace}
                else:
                    return {"success": False, "error": f"namespace not found: {namespace}"}
            else:
                # 清空默认命名空间
                if self.config.default_namespace in self.memory_store:
                    count = len(self.memory_store[self.config.default_namespace])
                    del self.memory_store[self.config.default_namespace]
                    self.save_to_file()
                    return {"success": True, "cleared": count, "namespace": self.config.default_namespace}
                else:
                    return {"success": True, "cleared": 0, "namespace": self.config.default_namespace}

    def export_json(self) -> str:
        """导出为 JSON"""
        with self._lock:
            data = {}
            for ns, entries in self.memory_store.items():
                data[ns] = {}
                for key, entry in entries.items():
                    data[ns][key] = asdict(entry)
            return json.dumps(data, ensure_ascii=False, indent=2)

    def import_json(self, json_str: str) -> Dict:
        """从 JSON 导入"""
        try:
            data = json.loads(json_str)
            with self._lock:
                for ns, entries in data.items():
                    if ns not in self.memory_store:
                        self.memory_store[ns] = {}
                    for key, entry_data in entries.items():
                        self.memory_store[ns][key] = MemoryEntry(
                            value=entry_data["value"],
                            timestamp=entry_data["timestamp"],
                            access_count=entry_data.get("access_count", 0)
                        )
                self.save_to_file()
            return {"success": True, "imported": sum(len(e) for e in data.values())}
        except Exception as e:
            return {"success": False, "error": str(e)}


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


def test_basic_store_retrieve():
    """测试 1: 基本存储与检索"""
    print("\n📋 测试 1: 基本存储与检索")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_1.json")
    store = MemoryStore(config)

    # 测试存储
    result = store.store("user_name", "张三")
    runner.record(
        "存储记忆成功",
        result["success"],
        str(result)
    )

    result = store.store("project", "Poyo")
    runner.record(
        "存储第二条记忆成功",
        result["success"]
    )

    # 测试检索
    result = store.retrieve("user_name")
    runner.record(
        "检索记忆成功",
        result["success"] and result["value"] == "张三",
        f"value={result.get('value')}"
    )

    # 测试访问计数
    store.retrieve("user_name")
    store.retrieve("user_name")
    result = store.retrieve("user_name")
    runner.record(
        "访问计数正确 (应为4次)",
        result["access_count"] == 4,
        f"access_count={result.get('access_count')}"
    )

    # 测试不存在的键
    result = store.retrieve("nonexistent")
    runner.record(
        "检索不存在的键返回错误",
        not result["success"]
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_namespace_isolation():
    """测试 2: 命名空间隔离"""
    print("\n📋 测试 2: 命名空间隔离")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_2.json")
    store = MemoryStore(config)

    # 在不同命名空间存储同名键
    store.store("config", "开发配置", namespace="dev")
    store.store("config", "生产配置", namespace="prod")

    # 检索验证
    result_dev = store.retrieve("config", namespace="dev")
    result_prod = store.retrieve("config", namespace="prod")

    runner.record(
        "dev 命名空间隔离正确",
        result_dev["value"] == "开发配置"
    )

    runner.record(
        "prod 命名空间隔离正确",
        result_prod["value"] == "生产配置"
    )

    # 测试默认命名空间
    store.store("config", "默认配置")
    result_default = store.retrieve("config")
    runner.record(
        "默认命名空间正确",
        result_default["value"] == "默认配置"
    )

    # 列出所有命名空间
    result = store.list()
    namespaces = set(m["namespace"] for m in result["memories"])
    runner.record(
        "命名空间列表正确",
        namespaces == {"dev", "prod", "default"},
        f"namespaces={namespaces}"
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_persistence():
    """测试 3: 持久化与恢复"""
    print("\n📋 测试 3: 持久化与恢复")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_3.json")

    # 存储数据
    store1 = MemoryStore(config)
    store1.store("key1", "value1")
    store1.store("key2", "value2", namespace="custom")
    store1.store("key3", "value3")

    # 获取统计
    stats1 = store1.get_stats()["stats"]

    # 创建新实例加载持久化数据
    store2 = MemoryStore(config)
    store2.load_from_file()

    # 验证恢复
    result = store2.retrieve("key1")
    runner.record(
        "恢复后检索 key1 成功",
        result["success"] and result["value"] == "value1"
    )

    result = store2.retrieve("key2", namespace="custom")
    runner.record(
        "恢复后检索自定义命名空间成功",
        result["success"] and result["value"] == "value2"
    )

    stats2 = store2.get_stats()["stats"]
    runner.record(
        "恢复后统计信息一致",
        stats2["total_keys"] == stats1["total_keys"],
        f"original={stats1['total_keys']}, recovered={stats2['total_keys']}"
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_memory_limits():
    """测试 4: 记忆限制"""
    print("\n📋 测试 4: 记忆限制")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(
        max_memory_size=100,  # 100 字节限制
        persistence_file="/tmp/poyo_memory_test_4.json"
    )
    store = MemoryStore(config)

    # 存储小数据
    result = store.store("small", "abc")
    runner.record(
        "小数据存储成功",
        result["success"]
    )

    # 尝试存储超大数据
    large_value = "x" * 200
    result = store.store("large", large_value)
    runner.record(
        "超大数据存储被拒绝",
        not result["success"] and "limit exceeded" in result.get("error", "")
    )

    # 验证统计
    stats = store.get_stats()["stats"]
    runner.record(
        "统计信息正确",
        stats["total_size"] < config.max_memory_size
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_search_functionality():
    """测试 5: 搜索功能"""
    print("\n📋 测试 5: 搜索功能")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_5.json")
    store = MemoryStore(config)

    # 存储测试数据
    store.store("api_key", "sk-1234567890")
    store.store("api_url", "https://api.example.com")
    store.store("database_url", "https://db.example.com")
    store.store("config_theme", "dark")
    store.store("config_lang", "zh-CN")

    # 搜索 "api"
    result = store.search("api")
    runner.record(
        "搜索 'api' 返回正确数量",
        result["total"] == 2,
        f"found={result['total']}"
    )

    # 搜索 "url"
    result = store.search("url")
    runner.record(
        "搜索 'url' 返回正确数量",
        result["total"] == 2
    )

    # 搜索 "config"
    result = store.search("config")
    runner.record(
        "搜索 'config' 返回正确数量",
        result["total"] == 2
    )

    # 搜索不存在的内容
    result = store.search("nonexistent")
    runner.record(
        "搜索不存在内容返回空",
        result["total"] == 0
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_forget_and_clear():
    """测试 6: 删除与清空"""
    print("\n📋 测试 6: 删除与清空")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_6.json")
    store = MemoryStore(config)

    # 存储数据
    store.store("key1", "value1")
    store.store("key2", "value2")
    store.store("key3", "value3", namespace="custom")

    # 删除单个记忆
    result = store.forget("key1")
    runner.record(
        "删除单个记忆成功",
        result["success"]
    )

    # 验证删除
    result = store.retrieve("key1")
    runner.record(
        "删除后检索失败",
        not result["success"]
    )

    # 清空默认命名空间
    result = store.clear()
    runner.record(
        "清空默认命名空间成功",
        result["success"] and result.get("cleared", 0) >= 1  # 至少清空了 key2
    )

    # 验证自定义命名空间仍存在
    result = store.retrieve("key3", namespace="custom")
    runner.record(
        "自定义命名空间未被清空",
        result["success"],
        str(result)
    )

    # 清空自定义命名空间
    result = store.clear(namespace="custom")
    runner.record(
        "清空自定义命名空间成功",
        result["success"],
        str(result)
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_concurrent_access():
    """测试 7: 并发访问"""
    print("\n📋 测试 7: 并发访问")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_7.json")
    store = MemoryStore(config)

    errors = []
    success_count = [0]

    def store_worker(thread_id: int):
        try:
            for i in range(20):
                result = store.store(f"thread_{thread_id}_key_{i}", f"value_{i}")
                if result["success"]:
                    success_count[0] += 1
        except Exception as e:
            errors.append(str(e))

    def retrieve_worker(thread_id: int):
        try:
            for i in range(20):
                store.retrieve(f"thread_{thread_id}_key_{i}")
        except Exception as e:
            errors.append(str(e))

    threads = []
    for i in range(5):
        t = threading.Thread(target=store_worker, args=(i,))
        threads.append(t)
        t.start()

    for t in threads:
        t.join()

    runner.record(
        "并发存储无错误",
        len(errors) == 0,
        f"errors={errors}"
    )

    runner.record(
        "所有存储操作成功",
        success_count[0] == 100,
        f"success_count={success_count[0]}"
    )

    # 验证数据完整性
    result = store.list()
    runner.record(
        "并发后数据完整",
        result["total"] == 100,
        f"total={result['total']}"
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_import_export():
    """测试 8: 导入导出"""
    print("\n📋 测试 8: 导入导出")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_8.json")
    store = MemoryStore(config)

    # 存储数据
    store.store("key1", "value1")
    store.store("key2", "value2")
    store.store("key3", "value3", namespace="custom")

    # 导出
    json_str = store.export_json()
    runner.record(
        "导出 JSON 成功",
        len(json_str) > 0
    )

    # 解析验证
    try:
        exported_data = json.loads(json_str)
        runner.record(
            "导出 JSON 格式正确",
            "default" in exported_data and "custom" in exported_data
        )
    except:
        runner.record("导出 JSON 格式正确", False)

    # 创建新 store 导入
    store2 = MemoryStore(config)
    result = store2.import_json(json_str)
    runner.record(
        "导入 JSON 成功",
        result["success"] and result["imported"] == 3
    )

    # 验证导入数据
    result = store2.retrieve("key2")
    runner.record(
        "导入后检索成功",
        result["success"] and result["value"] == "value2"
    )

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


def test_memory_lifecycle():
    """测试 9: 记忆生命周期管理"""
    print("\n📋 测试 9: 记忆生命周期管理")
    print("-" * 50)

    runner = TestRunner()
    config = MemoryConfig(persistence_file="/tmp/poyo_memory_test_9.json")
    store = MemoryStore(config)

    # 模拟完整的记忆生命周期
    print("\n  📝 阶段 1: 创建记忆")
    result = store.store("session_id", "sess-12345")
    runner.record("创建记忆成功", result["success"])

    print("  📖 阶段 2: 多次访问")
    for i in range(5):
        store.retrieve("session_id")

    result = store.retrieve("session_id")
    runner.record(
        "访问计数正确",
        result["access_count"] == 6,  # 5次 + 当前1次
        f"access_count={result['access_count']}"
    )

    print("  🔍 阶段 3: 搜索定位")
    result = store.search("session")
    runner.record("搜索定位成功", result["total"] == 1)

    print("  📊 阶段 4: 统计检查")
    stats = store.get_stats()["stats"]
    runner.record(
        "统计信息更新",
        stats["total_keys"] >= 1
    )

    print("  🗑️  阶段 5: 删除记忆")
    result = store.forget("session_id")
    runner.record("删除成功", result["success"])

    result = store.retrieve("session_id")
    runner.record("删除后无法检索", not result["success"])

    # 清理
    os.remove(config.persistence_file) if os.path.exists(config.persistence_file) else None

    runner.summary()
    return runner


# ==================== 主测试入口 ====================

def main():
    print("=" * 60)
    print("🧠 Poyo 长期记忆能力测试")
    print("=" * 60)

    all_runners = []

    # 运行所有测试
    all_runners.append(test_basic_store_retrieve())
    all_runners.append(test_namespace_isolation())
    all_runners.append(test_persistence())
    all_runners.append(test_memory_limits())
    all_runners.append(test_search_functionality())
    all_runners.append(test_forget_and_clear())
    all_runners.append(test_concurrent_access())
    all_runners.append(test_import_export())
    all_runners.append(test_memory_lifecycle())

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
        print("\n🎉 所有测试通过! Poyo 的长期记忆能力正常工作。")
    else:
        print(f"\n⚠️  有 {total_failed} 个测试失败，请检查相关功能。")

    # 清理测试文件
    for i in range(1, 10):
        test_file = f"/tmp/poyo_memory_test_{i}.json"
        if os.path.exists(test_file):
            os.remove(test_file)


if __name__ == "__main__":
    main()
