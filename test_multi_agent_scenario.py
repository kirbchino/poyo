#!/usr/bin/env python3
"""
Poyo 场景式多 Agent 模式测试

演示三种典型场景：
1. 软件开发流程 - 协调者分配任务，开发者编码，审核者审核，测试者验证
2. 研究报告生成 - 研究者收集信息，分析者分析，写作者撰写，校对者审核
3. DevOps 故障处理 - 监控者发现，运维者修复，审核者确认，验证者测试
"""

import json
import time
from dataclasses import dataclass, field, asdict
from typing import Dict, List, Optional
from enum import Enum
from datetime import datetime
import threading
from concurrent.futures import ThreadPoolExecutor, Future

# 颜色输出
class Colors:
    GREEN = '\033[92m'
    BLUE = '\033[94m'
    YELLOW = '\033[93m'
    CYAN = '\033[96m'
    RED = '\033[91m'
    MAGENTA = '\033[95m'
    BOLD = '\033[1m'
    END = '\033[0m'

def cprint(color: str, msg: str):
    print(f"{color}{msg}{Colors.END}")

# ==================== 类型定义 ====================

class AgentRole(Enum):
    COORDINATOR = "coordinator"  # 协调者
    WORKER = "worker"           # 工作者
    REVIEWER = "reviewer"       # 审核者
    VALIDATOR = "validator"     # 验证者
    RESEARCHER = "researcher"   # 研究者
    EXECUTOR = "executor"       # 执行者
    ANALYZER = "analyzer"       # 分析者
    WRITER = "writer"           # 写作者
    MONITOR = "monitor"         # 监控者
    DEVOPS = "devops"           # 运维者
    TESTER = "tester"           # 测试者

class TaskStatus(Enum):
    PENDING = "pending"
    ASSIGNED = "assigned"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"

class TeamState(Enum):
    CREATED = "created"
    ACTIVE = "active"
    COMPLETED = "completed"
    FAILED = "failed"

@dataclass
class Message:
    id: str
    from_agent: str
    to_agent: str
    type: str
    content: str
    timestamp: str = field(default_factory=lambda: datetime.now().isoformat())
    read: bool = False

@dataclass
class Task:
    id: str
    title: str
    description: str
    status: TaskStatus = TaskStatus.PENDING
    assigned_to: str = ""
    result: str = ""
    error: str = ""
    subtasks: List[str] = field(default_factory=list)
    created_at: str = field(default_factory=lambda: datetime.now().isoformat())

@dataclass
class Agent:
    id: str
    name: str
    role: AgentRole
    capabilities: List[str]
    status: str = "idle"
    current_task: str = ""
    inbox: List[Message] = field(default_factory=list)

    def receive_message(self, msg: Message):
        self.inbox.append(msg)

    def get_unread_messages(self) -> List[Message]:
        unread = [m for m in self.inbox if not m.read]
        for m in unread:
            m.read = True
        return unread

# ==================== 场景化 Agent 行为 ====================

class AgentBehavior:
    """Agent 行为基类"""

    def __init__(self, agent: Agent, team: 'Team'):
        self.agent = agent
        self.team = team

    def process_task(self, task: Task) -> str:
        """处理任务，返回结果"""
        raise NotImplementedError

class CoordinatorBehavior(AgentBehavior):
    """协调者行为"""

    def process_task(self, task: Task) -> str:
        cprint(Colors.CYAN, f"\n  🎯 [{self.agent.name}] 开始协调任务: {task.title}")

        # 分析任务，创建子任务
        subtasks = self._decompose_task(task)

        # 分配给合适的 agent
        assignments = self._assign_subtasks(subtasks)

        # 发送分配消息
        for agent_id, subtask in assignments.items():
            self.team.send_message(
                self.agent.id,
                agent_id,
                "task_assignment",
                f"请完成子任务: {subtask}"
            )

        return f"已分配 {len(assignments)} 个子任务"

    def _decompose_task(self, task: Task) -> List[Dict]:
        """任务分解"""
        decomposition = {
            "开发新功能": [
                {"title": "需求分析", "type": "research"},
                {"title": "编码实现", "type": "coding"},
                {"title": "代码审核", "type": "review"},
                {"title": "测试验证", "type": "test"},
            ],
            "修复Bug": [
                {"title": "问题定位", "type": "research"},
                {"title": "修复代码", "type": "coding"},
                {"title": "代码审核", "type": "review"},
                {"title": "验证修复", "type": "test"},
            ],
            "生成报告": [
                {"title": "收集资料", "type": "research"},
                {"title": "分析数据", "type": "analysis"},
                {"title": "撰写报告", "type": "writing"},
                {"title": "校对审核", "type": "review"},
            ],
            "处理故障": [
                {"title": "故障诊断", "type": "diagnosis"},
                {"title": "执行修复", "type": "fix"},
                {"title": "变更审核", "type": "review"},
                {"title": "验证恢复", "type": "verify"},
            ],
        }

        for key, subs in decomposition.items():
            if key in task.title:
                return subs

        return [{"title": "执行任务", "type": "general"}]

    def _assign_subtasks(self, subtasks: List[Dict]) -> Dict[str, str]:
        """分配子任务"""
        assignments = {}

        for agent_id, agent in self.team.agents.items():
            if agent.id == self.agent.id:
                continue

            for subtask in subtasks:
                subtask_type = subtask.get("type", "")

                # 匹配能力
                if self._can_handle(agent, subtask_type):
                    if agent.id not in assignments:
                        assignments[agent.id] = subtask["title"]
                        break

        return assignments

    def _can_handle(self, agent: Agent, task_type: str) -> bool:
        capability_map = {
            "research": ["research", "analyze"],
            "coding": ["code", "develop", "implement"],
            "review": ["review", "audit", "check"],
            "test": ["test", "verify", "validate"],
            "analysis": ["analyze", "process"],
            "writing": ["write", "document"],
            "diagnosis": ["diagnose", "monitor"],
            "fix": ["fix", "repair", "devops"],
            "verify": ["verify", "test", "validate"],
        }

        required = capability_map.get(task_type, [])
        return any(cap in agent.capabilities for cap in required)

class WorkerBehavior(AgentBehavior):
    """工作者行为（开发/执行）"""

    def process_task(self, task: Task) -> str:
        cprint(Colors.GREEN, f"\n  💻 [{self.agent.name}] 执行任务: {task.title}")

        steps = [
            "读取相关代码...",
            "理解上下文...",
            "编写实现...",
            "自我检查...",
        ]

        for step in steps:
            cprint(Colors.BLUE, f"    → {step}")
            time.sleep(0.3)

        result = f"完成: {task.title} 的实现"
        cprint(Colors.GREEN, f"    ✓ {result}")

        # 通知完成
        self.team.send_message(
            self.agent.id,
            self._find_reviewer(),
            "review_request",
            f"请审核: {task.title}"
        )

        return result

    def _find_reviewer(self) -> str:
        for agent_id, agent in self.team.agents.items():
            if agent.role == AgentRole.REVIEWER:
                return agent_id
        return ""

class ReviewerBehavior(AgentBehavior):
    """审核者行为"""

    def process_task(self, task: Task) -> str:
        cprint(Colors.YELLOW, f"\n  🔍 [{self.agent.name}] 审核任务: {task.title}")

        checks = [
            "检查代码风格...",
            "检查逻辑正确性...",
            "检查安全性...",
            "检查测试覆盖...",
        ]

        issues = []
        for check in checks:
            cprint(Colors.BLUE, f"    → {check}")
            time.sleep(0.2)

        # 模拟审核结果
        result = f"审核通过: {task.title}"
        cprint(Colors.YELLOW, f"    ✓ {result}")

        # 通知验证者
        validator = self._find_validator()
        if validator:
            self.team.send_message(
                self.agent.id,
                validator,
                "validate_request",
                f"请验证: {task.title}"
            )

        return result

    def _find_validator(self) -> str:
        for agent_id, agent in self.team.agents.items():
            if agent.role == AgentRole.VALIDATOR or agent.role == AgentRole.TESTER:
                return agent_id
        return ""

class ValidatorBehavior(AgentBehavior):
    """验证者行为"""

    def process_task(self, task: Task) -> str:
        cprint(Colors.MAGENTA, f"\n  ✅ [{self.agent.name}] 验证任务: {task.title}")

        tests = [
            "运行单元测试...",
            "运行集成测试...",
            "验证功能完整性...",
        ]

        for test in tests:
            cprint(Colors.BLUE, f"    → {test}")
            time.sleep(0.2)

        result = f"验证通过: {task.title}"
        cprint(Colors.MAGENTA, f"    ✓ {result}")

        return result

class ResearcherBehavior(AgentBehavior):
    """研究者行为"""

    def process_task(self, task: Task) -> str:
        cprint(Colors.CYAN, f"\n  📚 [{self.agent.name}] 研究任务: {task.title}")

        steps = [
            "搜索相关资料...",
            "提取关键信息...",
            "整理研究成果...",
        ]

        for step in steps:
            cprint(Colors.BLUE, f"    → {step}")
            time.sleep(0.2)

        result = f"研究完成: {task.title}"
        cprint(Colors.CYAN, f"    ✓ {result}")

        return result

class MonitorBehavior(AgentBehavior):
    """监控者行为"""

    def process_task(self, task: Task) -> str:
        cprint(Colors.RED, f"\n  🚨 [{self.agent.name}] 监控发现异常: {task.title}")

        steps = [
            "收集告警信息...",
            "分析问题根因...",
            "确定故障级别...",
        ]

        for step in steps:
            cprint(Colors.BLUE, f"    → {step}")
            time.sleep(0.2)

        # 通知运维者
        devops = self._find_devops()
        if devops:
            self.team.send_message(
                self.agent.id,
                devops,
                "incident_report",
                f"故障报告: {task.title}"
            )

        result = f"故障已诊断: {task.title}"
        cprint(Colors.RED, f"    ✓ {result}")

        return result

    def _find_devops(self) -> str:
        for agent_id, agent in self.team.agents.items():
            if agent.role == AgentRole.DEVOPS:
                return agent_id
        return ""

class DevOpsBehavior(AgentBehavior):
    """运维者行为"""

    def process_task(self, task: Task) -> str:
        cprint(Colors.RED, f"\n  🔧 [{self.agent.name}] 运维任务: {task.title}")

        steps = [
            "诊断问题...",
            "制定修复方案...",
            "执行修复...",
            "验证恢复...",
        ]

        for step in steps:
            cprint(Colors.BLUE, f"    → {step}")
            time.sleep(0.2)

        result = f"修复完成: {task.title}"
        cprint(Colors.RED, f"    ✓ {result}")

        return result

# ==================== 团队系统 ====================

@dataclass
class Team:
    id: str
    name: str
    description: str
    state: TeamState = TeamState.CREATED
    agents: Dict[str, Agent] = field(default_factory=dict)
    tasks: Dict[str, Task] = field(default_factory=dict)
    messages: List[Message] = field(default_factory=list)
    created_at: str = field(default_factory=lambda: datetime.now().isoformat())

    _agent_behaviors: Dict[str, AgentBehavior] = field(default_factory=dict, repr=False)

    def add_agent(self, agent: Agent, behavior: AgentBehavior = None):
        self.agents[agent.id] = agent
        if behavior:
            self._agent_behaviors[agent.id] = behavior

    def create_task(self, task: Task):
        self.tasks[task.id] = task

    def send_message(self, from_agent: str, to_agent: str, msg_type: str, content: str):
        msg = Message(
            id=f"msg-{len(self.messages)}",
            from_agent=from_agent,
            to_agent=to_agent,
            type=msg_type,
            content=content,
        )
        self.messages.append(msg)

        if to_agent and to_agent in self.agents:
            self.agents[to_agent].receive_message(msg)

    def get_agent_behavior(self, agent: Agent) -> AgentBehavior:
        if agent.id in self._agent_behaviors:
            return self._agent_behaviors[agent.id]

        # 默认行为映射
        behavior_map = {
            AgentRole.COORDINATOR: CoordinatorBehavior,
            AgentRole.WORKER: WorkerBehavior,
            AgentRole.REVIEWER: ReviewerBehavior,
            AgentRole.VALIDATOR: ValidatorBehavior,
            AgentRole.RESEARCHER: ResearcherBehavior,
            AgentRole.EXECUTOR: WorkerBehavior,
            AgentRole.DEVOPS: DevOpsBehavior,
            AgentRole.TESTER: ValidatorBehavior,
            AgentRole.MONITOR: MonitorBehavior,
        }

        behavior_class = behavior_map.get(agent.role, WorkerBehavior)
        return behavior_class(agent, self)

# ==================== 场景工厂 ====================

class ScenarioFactory:
    """场景工厂 - 创建预定义的多 Agent 场景"""

    @staticmethod
    def create_dev_team() -> Team:
        """创建软件开发团队"""
        team = Team(
            id="team-dev-001",
            name="开发团队",
            description="负责软件开发全流程"
        )

        # 协调者
        coordinator = Agent(
            id="agent-coord",
            name="协调者-卡比",
            role=AgentRole.COORDINATOR,
            capabilities=["coordinate", "plan", "assign"],
        )
        team.add_agent(coordinator)

        # 开发者
        developer = Agent(
            id="agent-dev",
            name="开发者-剑士卡比",
            role=AgentRole.WORKER,
            capabilities=["code", "develop", "implement", "debug"],
        )
        team.add_agent(developer)

        # 审核者
        reviewer = Agent(
            id="agent-review",
            name="审核者-刀片卡比",
            role=AgentRole.REVIEWER,
            capabilities=["review", "audit", "check"],
        )
        team.add_agent(reviewer)

        # 测试者
        tester = Agent(
            id="agent-test",
            name="测试者-闪电卡比",
            role=AgentRole.TESTER,
            capabilities=["test", "verify", "validate"],
        )
        team.add_agent(tester)

        return team

    @staticmethod
    def create_research_team() -> Team:
        """创建研究团队"""
        team = Team(
            id="team-research-001",
            name="研究团队",
            description="负责研究和报告生成"
        )

        # 协调者
        coordinator = Agent(
            id="agent-research-coord",
            name="研究协调者",
            role=AgentRole.COORDINATOR,
            capabilities=["coordinate", "plan"],
        )
        team.add_agent(coordinator)

        # 研究者
        researcher = Agent(
            id="agent-research",
            name="研究者-吸入卡比",
            role=AgentRole.RESEARCHER,
            capabilities=["research", "search", "collect"],
        )
        team.add_agent(researcher)

        # 分析者
        analyzer = Agent(
            id="agent-analyze",
            name="分析者-火焰卡比",
            role=AgentRole.ANALYZER,
            capabilities=["analyze", "process", "summarize"],
        )
        team.add_agent(analyzer)

        # 写作者
        writer = Agent(
            id="agent-write",
            name="写作者-石头卡比",
            role=AgentRole.WRITER,
            capabilities=["write", "document", "format"],
        )
        team.add_agent(writer)

        return team

    @staticmethod
    def create_devops_team() -> Team:
        """创建 DevOps 团队"""
        team = Team(
            id="team-devops-001",
            name="运维团队",
            description="负责系统运维和故障处理"
        )

        # 监控者
        monitor = Agent(
            id="agent-monitor",
            name="监控者-光束卡比",
            role=AgentRole.MONITOR,
            capabilities=["monitor", "detect", "alert"],
        )
        team.add_agent(monitor)

        # 运维者
        devops = Agent(
            id="agent-devops",
            name="运维者-忍者卡比",
            role=AgentRole.DEVOPS,
            capabilities=["fix", "repair", "deploy", "configure"],
        )
        team.add_agent(devops)

        # 审核者
        reviewer = Agent(
            id="agent-ops-review",
            name="变更审核者",
            role=AgentRole.REVIEWER,
            capabilities=["review", "approve"],
        )
        team.add_agent(reviewer)

        # 验证者
        validator = Agent(
            id="agent-ops-validate",
            name="恢复验证者",
            role=AgentRole.VALIDATOR,
            capabilities=["verify", "test", "validate"],
        )
        team.add_agent(validator)

        return team

# ==================== 场景执行器 ====================

class ScenarioRunner:
    """场景执行器"""

    def __init__(self, team: Team):
        self.team = team
        self.task_results: Dict[str, str] = {}

    def run_scenario(self, scenario_name: str, main_task: str):
        """运行场景"""
        print(f"\n{'='*60}")
        cprint(Colors.BOLD, f"🎬 场景: {scenario_name}")
        print(f"{'='*60}")

        cprint(Colors.CYAN, f"\n📋 主任务: {main_task}")
        cprint(Colors.CYAN, f"👥 团队: {self.team.name}")
        cprint(Colors.CYAN, f"🤖 成员: {', '.join(a.name for a in self.team.agents.values())}")

        # 创建主任务
        task = Task(
            id="task-main",
            title=main_task,
            description=f"完成 {main_task}",
        )
        self.team.create_task(task)

        # 找到协调者
        coordinator = None
        for agent in self.team.agents.values():
            if agent.role == AgentRole.COORDINATOR or agent.role == AgentRole.MONITOR:
                coordinator = agent
                break

        if coordinator:
            # 协调者处理任务
            behavior = self.team.get_agent_behavior(coordinator)
            result = behavior.process_task(task)
            self.task_results[task.id] = result

            # 处理消息并执行子任务
            self._process_messages()

        print(f"\n{'='*60}")
        cprint(Colors.GREEN, f"🎉 场景完成: {scenario_name}")
        print(f"{'='*60}")

        return self.task_results

    def _process_messages(self):
        """处理消息队列"""
        max_rounds = 5

        for _ in range(max_rounds):
            # 收集所有未读消息
            all_messages = []
            for agent in self.team.agents.values():
                all_messages.extend(agent.get_unread_messages())

            if not all_messages:
                break

            # 处理消息
            for msg in all_messages:
                if msg.type == "task_assignment":
                    # 创建子任务并执行
                    self._execute_subtask(msg.to_agent, msg.content)

                elif msg.type == "review_request":
                    # 审核者处理
                    self._handle_review(msg.to_agent, msg.content)

                elif msg.type == "validate_request":
                    # 验证者处理
                    self._handle_validation(msg.to_agent, msg.content)

                elif msg.type == "incident_report":
                    # DevOps 处理故障报告
                    self._execute_subtask(msg.to_agent, msg.content)

                elif msg.type == "change_request":
                    # 审核变更请求
                    self._handle_review(msg.to_agent, msg.content)

    def _execute_subtask(self, agent_id: str, task_content: str):
        """执行子任务"""
        if agent_id not in self.team.agents:
            return

        agent = self.team.agents[agent_id]
        task = Task(
            id=f"subtask-{agent_id}",
            title=task_content,
            description=task_content,
            status=TaskStatus.RUNNING,
        )

        behavior = self.team.get_agent_behavior(agent)
        result = behavior.process_task(task)
        self.task_results[task.id] = result

    def _handle_review(self, agent_id: str, content: str):
        """处理审核请求"""
        if agent_id not in self.team.agents:
            return

        agent = self.team.agents[agent_id]
        task = Task(
            id=f"review-{agent_id}",
            title=f"审核: {content}",
            description=content,
        )

        behavior = self.team.get_agent_behavior(agent)
        result = behavior.process_task(task)
        self.task_results[task.id] = result

    def _handle_validation(self, agent_id: str, content: str):
        """处理验证请求"""
        if agent_id not in self.team.agents:
            return

        agent = self.team.agents[agent_id]
        task = Task(
            id=f"validate-{agent_id}",
            title=f"验证: {content}",
            description=content,
        )

        behavior = self.team.get_agent_behavior(agent)
        result = behavior.process_task(task)
        self.task_results[task.id] = result

# ==================== 测试函数 ====================

def test_software_dev_scenario():
    """测试软件开发场景"""
    print(f"\n{Colors.CYAN}{'='*60}{Colors.END}")
    cprint(Colors.BOLD, "🧪 场景一: 软件开发流程")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}")

    team = ScenarioFactory.create_dev_team()
    runner = ScenarioRunner(team)

    results = runner.run_scenario(
        "开发新功能",
        "开发用户登录功能"
    )

    return len(results) > 0

def test_research_scenario():
    """测试研究报告场景"""
    print(f"\n{Colors.CYAN}{'='*60}{Colors.END}")
    cprint(Colors.BOLD, "🧪 场景二: 研究报告生成")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}")

    team = ScenarioFactory.create_research_team()
    runner = ScenarioRunner(team)

    results = runner.run_scenario(
        "生成报告",
        "生成AI技术调研报告"
    )

    return len(results) > 0

def test_devops_scenario():
    """测试 DevOps 场景"""
    print(f"\n{Colors.CYAN}{'='*60}{Colors.END}")
    cprint(Colors.BOLD, "🧪 场景三: DevOps 故障处理")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}")

    team = ScenarioFactory.create_devops_team()
    runner = ScenarioRunner(team)

    results = runner.run_scenario(
        "处理故障",
        "修复数据库连接超时问题"
    )

    return len(results) > 0

def test_agent_collaboration():
    """测试 Agent 协作"""
    print(f"\n{Colors.CYAN}{'='*60}{Colors.END}")
    cprint(Colors.BOLD, "🧪 测试: Agent 消息协作")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}")

    team = ScenarioFactory.create_dev_team()

    # 发送消息
    team.send_message("agent-coord", "agent-dev", "task", "开始编码")
    team.send_message("agent-dev", "agent-review", "request", "代码审核")
    team.send_message("agent-review", "agent-test", "request", "测试验证")

    # 检查消息
    dev_msgs = team.agents["agent-dev"].get_unread_messages()
    review_msgs = team.agents["agent-review"].get_unread_messages()
    test_msgs = team.agents["agent-test"].get_unread_messages()

    print(f"\n  📬 开发者收到消息: {len(dev_msgs)}")
    print(f"  📬 审核者收到消息: {len(review_msgs)}")
    print(f"  📬 测试者收到消息: {len(test_msgs)}")

    return len(dev_msgs) > 0 and len(review_msgs) > 0 and len(test_msgs) > 0

def main():
    print(f"\n{Colors.GREEN}{'='*60}{Colors.END}")
    cprint(Colors.BOLD, "🤖 Poyo 场景式多 Agent 模式测试")
    print(f"{Colors.GREEN}{'='*60}{Colors.END}")

    results = []

    # 运行测试
    results.append(("软件开发场景", test_software_dev_scenario()))
    results.append(("研究报告场景", test_research_scenario()))
    results.append(("DevOps场景", test_devops_scenario()))
    results.append(("Agent协作测试", test_agent_collaboration()))

    # 输出结果
    print(f"\n{Colors.CYAN}{'='*60}{Colors.END}")
    cprint(Colors.BOLD, "📊 测试结果汇总")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}")

    all_passed = True
    for name, passed in results:
        status = "✅ PASS" if passed else "❌ FAIL"
        color = Colors.GREEN if passed else Colors.RED
        cprint(color, f"  {status}: {name}")
        if not passed:
            all_passed = False

    print(f"\n{Colors.CYAN}{'='*60}{Colors.END}")
    if all_passed:
        cprint(Colors.GREEN, "🎉 所有场景测试通过!")
    else:
        cprint(Colors.RED, "⚠️ 部分场景测试失败")
    print(f"{Colors.CYAN}{'='*60}{Colors.END}\n")

    return all_passed

if __name__ == "__main__":
    import sys
    sys.exit(0 if main() else 1)
