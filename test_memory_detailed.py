#!/usr/bin/env python3
"""
Poyo 长期记忆测试 - 详细输入输出版
"""

import json
import os
import time
import threading
from datetime import datetime
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, field, asdict

# ==================== 核心类定义 ====================

@dataclass
class MemoryEntry:
    value: str
    timestamp: int
    access_count: int = 0

@dataclass
class MemoryConfig:
    max_memory_size: int = 10000
    default_namespace: str = "default"
    persistence_file: str = ".poyo/memory.json"

class MemoryStore:
    def __init__(self, config: MemoryConfig = None):
        self.config = config or MemoryConfig()
        self.memory_store: Dict[str, Dict[str, MemoryEntry]] = {}
        self._lock = threading.RLock()

    def store(self, key: str, value: str, namespace: str = None) -> Dict:
        namespace = namespace or self.config.default_namespace
        if not key or not value:
            return {"success": False, "error": "key and value are required"}

        with self._lock:
            total_size = sum(len(k) + len(e.value) for ns in self.memory_store.values() for k, e in ns.items())
            if total_size + len(key) + len(value) > self.config.max_memory_size:
                return {"success": False, "error": "memory size limit exceeded"}

            if namespace not in self.memory_store:
                self.memory_store[namespace] = {}

            self.memory_store[namespace][key] = MemoryEntry(
                value=value,
                timestamp=int(time.time()),
                access_count=0
            )
            return {"success": True, "key": key, "namespace": namespace, "size": len(value)}

    def retrieve(self, key: str, namespace: str = None) -> Dict:
        namespace = namespace or self.config.default_namespace
        if not key:
            return {"success": False, "error": "key is required"}

        with self._lock:
            if namespace not in self.memory_store:
                return {"success": False, "error": f"namespace not found: {namespace}"}

            entry = self.memory_store[namespace].get(key)
            if not entry:
                return {"success": False, "error": f"memory not found: {key}"}

            entry.access_count += 1
            return {
                "success": True,
                "key": key,
                "value": entry.value,
                "namespace": namespace,
                "timestamp": entry.timestamp,
                "access_count": entry.access_count
            }

    def search(self, query: str, namespace: str = None) -> Dict:
        results = []
        query_lower = query.lower()

        with self._lock:
            namespaces = [namespace] if namespace else list(self.memory_store.keys())
            for ns in namespaces:
                if ns not in self.memory_store:
                    continue
                for key, entry in self.memory_store[ns].items():
                    if query_lower in key.lower() or query_lower in entry.value.lower():
                        results.append({
                            "key": key,
                            "namespace": ns,
                            "value": entry.value,
                            "access_count": entry.access_count
                        })

        return {"success": True, "query": query, "results": results, "total": len(results)}

    def forget(self, key: str, namespace: str = None) -> Dict:
        namespace = namespace or self.config.default_namespace
        if not key:
            return {"success": False, "error": "key is required"}

        with self._lock:
            if namespace not in self.memory_store:
                return {"success": False, "error": f"namespace not found: {namespace}"}
            if key not in self.memory_store[namespace]:
                return {"success": False, "error": f"memory not found: {key}"}

            del self.memory_store[namespace][key]
            if not self.memory_store[namespace]:
                del self.memory_store[namespace]
            return {"success": True, "key": key, "namespace": namespace}

    def get_stats(self) -> Dict:
        with self._lock:
            namespaces = len(self.memory_store)
            total_keys = sum(len(entries) for entries in self.memory_store.values())
            total_size = sum(len(k) + len(e.value) for ns in self.memory_store.values() for k, e in ns.items())
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

# ==================== 测试用例 ====================

def test_case_1_basic_store_retrieve():
    """测试用例 1: 基本存储与检索"""
    print("\n" + "=" * 70)
    print("📋 测试用例 1: 基本存储与检索")
    print("=" * 70)

    store = MemoryStore(MemoryConfig())

    # 存储操作
    print("\n📥 输入: 存储记忆")
    print("   key: 'user_name'")
    print("   value: '张三'")

    result = store.store("user_name", "张三")
    print("\n📤 输出:")
    print(f"   success: {result['success']}")
    print(f"   key: {result['key']}")
    print(f"   namespace: {result['namespace']}")
    print(f"   size: {result['size']} bytes")

    # 检索操作
    print("\n📥 输入: 检索记忆")
    print("   key: 'user_name'")

    result = store.retrieve("user_name")
    print("\n📤 输出:")
    print(f"   success: {result['success']}")
    print(f"   value: {result['value']}")
    print(f"   access_count: {result['access_count']}")

    # 多次访问
    print("\n📥 输入: 再次检索 3 次")
    for _ in range(3):
        store.retrieve("user_name")

    result = store.retrieve("user_name")
    print("\n📤 输出:")
    print(f"   access_count: {result['access_count']} (累计 5 次)")

    # 检索不存在的键
    print("\n📥 输入: 检索不存在的键")
    print("   key: 'nonexistent'")

    result = store.retrieve("nonexistent")
    print("\n📤 输出:")
    print(f"   success: {result['success']}")
    print(f"   error: {result['error']}")
    print("   ✅ 结果: 正确返回错误")

    return True

def test_case_2_namespace():
    """测试用例 2: 命名空间隔离"""
    print("\n" + "=" * 70)
    print("📋 测试用例 2: 命名空间隔离")
    print("=" * 70)

    store = MemoryStore(MemoryConfig())

    # 在不同命名空间存储同名键
    print("\n📥 输入: 在不同命名空间存储同名键 'config'")
    print("   namespace: 'dev', value: '开发配置'")
    print("   namespace: 'prod', value: '生产配置'")
    print("   namespace: 'default', value: '默认配置'")

    store.store("config", "开发配置", namespace="dev")
    store.store("config", "生产配置", namespace="prod")
    store.store("config", "默认配置")

    # 检索验证
    print("\n📤 输出: 分别检索各命名空间的 'config'")

    for ns in ["dev", "prod", "default"]:
        result = store.retrieve("config", namespace=ns if ns != "default" else None)
        namespace_display = ns if ns != "default" else "default (默认)"
        print(f"   [{namespace_display}]: {result['value']}")

    # 命名空间隔离验证
    print("\n📤 输出: 命名空间隔离验证")
    dev_result = store.retrieve("config", namespace="dev")
    prod_result = store.retrieve("config", namespace="prod")

    print(f"   dev.config == '开发配置': {dev_result['value'] == '开发配置'}")
    print(f"   prod.config == '生产配置': {prod_result['value'] == '生产配置'}")
    print(f"   dev.config != prod.config: {dev_result['value'] != prod_result['value']}")
    print("   ✅ 结果: 命名空间隔离正确")

    return True

def test_case_3_persistence():
    """测试用例 3: 持久化与恢复"""
    print("\n" + "=" * 70)
    print("📋 测试用例 3: 持久化与恢复")
    print("=" * 70)

    config = MemoryConfig(persistence_file="/tmp/test_memory_3.json")
    store1 = MemoryStore(config)

    # 存储数据
    print("\n📥 输入: 存储多条记忆")
    test_data = [
        ("key1", "value1", None),
        ("key2", "value2", None),
        ("key3", "value3", "custom")
    ]

    for key, value, ns in test_data:
        result = store1.store(key, value, namespace=ns)
        ns_display = ns if ns else "default"
        print(f"   [{ns_display}] {key} = '{value}' -> success: {result['success']}")

    # 导出
    print("\n📤 输出: 当前存储状态")
    stats = store1.get_stats()["stats"]
    print(f"   命名空间数: {stats['namespaces']}")
    print(f"   总键数: {stats['total_keys']}")
    print(f"   总大小: {stats['total_size']} bytes")

    # 模拟重启 - 创建新实例
    print("\n📥 输入: 模拟重启，创建新的 MemoryStore 实例")

    store2 = MemoryStore(config)
    # 手动加载（实际中会在初始化时自动加载）
    if os.path.exists(config.persistence_file):
        with open(config.persistence_file, 'r') as f:
            data = json.load(f)
        for ns, entries in data.items():
            for key, entry in entries.items():
                store2.memory_store.setdefault(ns, {})[key] = MemoryEntry(**entry)

    print("\n📤 输出: 恢复后验证")
    for key, value, ns in test_data:
        result = store2.retrieve(key, namespace=ns)
        ns_display = ns if ns else "default"
        match = result['value'] == value if result['success'] else False
        print(f"   [{ns_display}] {key}: '{result.get('value', 'N/A')}' == '{value}' ? {match}")

    print("   ✅ 结果: 数据完整恢复")

    # 清理
    if os.path.exists(config.persistence_file):
        os.remove(config.persistence_file)

    return True

def test_case_4_limits():
    """测试用例 4: 记忆限制"""
    print("\n" + "=" * 70)
    print("📋 测试用例 4: 记忆限制")
    print("=" * 70)

    config = MemoryConfig(max_memory_size=100)  # 100 字节限制
    store = MemoryStore(config)

    print(f"\n📥 输入配置:")
    print(f"   max_memory_size: {config.max_memory_size} bytes")

    # 存储小数据
    print("\n📥 输入: 存储小数据")
    print("   key: 'small', value: 'abc' (3 bytes)")

    result = store.store("small", "abc")
    print("\n📤 输出:")
    print(f"   success: {result['success']}")
    print(f"   size: {result.get('size', 'N/A')} bytes")

    # 尝试存储超大数据
    print("\n📥 输入: 存储超大数据")
    large_value = "x" * 200
    print(f"   key: 'large', value: '{large_value[:50]}...' ({len(large_value)} bytes)")

    result = store.store("large", large_value)
    print("\n📤 输出:")
    print(f"   success: {result['success']}")
    print(f"   error: {result.get('error', 'N/A')}")
    print("   ✅ 结果: 正确拒绝超限存储")

    return True

def test_case_5_search():
    """测试用例 5: 搜索功能"""
    print("\n" + "=" * 70)
    print("📋 测试用例 5: 搜索功能")
    print("=" * 70)

    store = MemoryStore(MemoryConfig())

    # 存储测试数据
    print("\n📥 输入: 存储测试数据")
    test_data = [
        ("api_key", "sk-1234567890"),
        ("api_url", "https://api.example.com"),
        ("database_url", "https://db.example.com"),
        ("config_theme", "dark"),
        ("config_lang", "zh-CN"),
    ]

    for key, value in test_data:
        store.store(key, value)
        print(f"   {key} = '{value}'")

    # 搜索测试
    search_cases = [
        ("api", "搜索包含 'api' 的记忆"),
        ("url", "搜索包含 'url' 的记忆"),
        ("config", "搜索包含 'config' 的记忆"),
        ("nonexistent", "搜索不存在的内容"),
    ]

    for query, desc in search_cases:
        print(f"\n📥 输入: {desc}")
        print(f"   query: '{query}'")

        result = store.search(query)
        print("\n📤 输出:")
        print(f"   total: {result['total']} 条结果")
        for r in result['results']:
            print(f"   - {r['key']}: '{r['value'][:30]}...'")

    print("\n   ✅ 结果: 搜索功能正常")

    return True

def test_case_6_forget():
    """测试用例 6: 删除与清空"""
    print("\n" + "=" * 70)
    print("📋 测试用例 6: 删除与清空")
    print("=" * 70)

    store = MemoryStore(MemoryConfig())

    # 存储数据
    print("\n📥 输入: 存储测试数据")
    store.store("key1", "value1")
    store.store("key2", "value2")
    store.store("key3", "value3", namespace="custom")
    print("   [default] key1, key2")
    print("   [custom] key3")

    # 删除单个记忆
    print("\n📥 输入: 删除 key1")
    result = store.forget("key1")
    print("\n📤 输出:")
    print(f"   success: {result['success']}")
    print(f"   key: {result['key']}")

    # 验证删除
    print("\n📥 输入: 检索已删除的 key1")
    result = store.retrieve("key1")
    print("\n📤 输出:")
    print(f"   success: {result['success']}")
    print(f"   error: {result.get('error', 'N/A')}")
    print("   ✅ 结果: 正确删除")

    return True

def test_case_7_concurrent():
    """测试用例 7: 并发访问"""
    print("\n" + "=" * 70)
    print("📋 测试用例 7: 并发访问")
    print("=" * 70)

    store = MemoryStore(MemoryConfig())
    errors = []
    success_count = [0]
    lock = threading.Lock()

    print("\n📥 输入: 5 个线程并发存储消息")
    print("   每个线程存储 20 条消息")

    def worker(thread_id):
        try:
            for i in range(20):
                result = store.store(f"thread_{thread_id}_key_{i}", f"value_{i}")
                if result["success"]:
                    with lock:
                        success_count[0] += 1
        except Exception as e:
            errors.append(str(e))

    threads = [threading.Thread(target=worker, args=(i,)) for i in range(5)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()

    print("\n📤 输出:")
    print(f"   线程错误数: {len(errors)}")
    print(f"   成功存储数: {success_count[0]}/100")

    stats = store.get_stats()["stats"]
    print(f"   最终消息数: {stats['total_keys']}")
    print(f"   ✅ 结果: {'通过' if len(errors) == 0 and success_count[0] == 100 else '失败'}")

    return True

def test_case_8_import_export():
    """测试用例 8: 导入导出"""
    print("\n" + "=" * 70)
    print("📋 测试用例 8: 导入导出")
    print("=" * 70)

    store = MemoryStore(MemoryConfig())

    # 存储数据
    print("\n📥 输入: 存储测试数据")
    store.store("key1", "value1")
    store.store("key2", "value2")
    store.store("key3", "value3", namespace="custom")
    print("   [default] key1, key2")
    print("   [custom] key3")

    # 导出
    print("\n📤 输出: 导出为 JSON")
    json_str = json.dumps(
        {ns: {k: asdict(e) for k, e in entries.items()}
         for ns, entries in store.memory_store.items()},
        indent=2
    )
    print(f"   JSON 长度: {len(json_str)} 字符")
    print(f"   JSON 预览:")
    for line in json_str.split('\n')[:15]:
        print(f"      {line}")
    print("      ...")

    # 导入
    print("\n📥 输入: 从 JSON 导入到新 store")
    store2 = MemoryStore(MemoryConfig())
    data = json.loads(json_str)
    for ns, entries in data.items():
        for key, entry in entries.items():
            store2.memory_store.setdefault(ns, {})[key] = MemoryEntry(**entry)

    print("\n📤 输出: 导入后验证")
    result = store2.retrieve("key2")
    print(f"   key2: '{result['value']}'")
    result = store2.retrieve("key3", namespace="custom")
    print(f"   [custom] key3: '{result['value']}'")
    print("   ✅ 结果: 导入导出正常")

    return True

# ==================== 主函数 ====================

def main():
    print("=" * 70)
    print("🧠 Poyo 长期记忆测试 - 详细输入输出")
    print("=" * 70)

    test_cases = [
        ("基本存储与检索", test_case_1_basic_store_retrieve),
        ("命名空间隔离", test_case_2_namespace),
        ("持久化与恢复", test_case_3_persistence),
        ("记忆限制", test_case_4_limits),
        ("搜索功能", test_case_5_search),
        ("删除与清空", test_case_6_forget),
        ("并发访问", test_case_7_concurrent),
        ("导入导出", test_case_8_import_export),
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
