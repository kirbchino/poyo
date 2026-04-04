#!/usr/bin/env node
/**
 * 🔌 MCP (Model Context Protocol) Server Example
 * 展示 MCP 服务器的所有能力
 *
 * MCP 协议支持:
 * - Tools: 可调用的工具
 * - Resources: 可访问的资源
 * - Prompts: 可使用的提示模板
 * - Sampling: 请求 LLM 采样
 * - Logging: 日志记录
 */

const readline = require('readline');

// ═══════════════════════════════════════════════════════
// 🔧 MCP Server Implementation
// ═══════════════════════════════════════════════════════

class MCPServer {
    constructor() {
        this.tools = this.defineTools();
        this.resources = this.defineResources();
        this.prompts = this.definePrompts();
        this.requestHandlers = this.defineRequestHandlers();

        this.rl = readline.createInterface({
            input: process.stdin,
            output: process.stdout,
            terminal: false
        });

        this.log('MCP Server initialized');
    }

    /**
     * 定义工具
     */
    defineTools() {
        return [
            {
                name: 'mcp_hello',
                description: '💚 MCP Hello 工具 - 说你好!',
                inputSchema: {
                    type: 'object',
                    properties: {
                        name: {
                            type: 'string',
                            description: '要问候的名字'
                        }
                    },
                    required: ['name']
                }
            },
            {
                name: 'mcp_calculate',
                description: '🧮 MCP 计算工具 - 执行数学运算',
                inputSchema: {
                    type: 'object',
                    properties: {
                        a: { type: 'number', description: '第一个数字' },
                        b: { type: 'number', description: '第二个数字' },
                        operation: {
                            type: 'string',
                            enum: ['add', 'sub', 'mul', 'div'],
                            description: '操作类型'
                        }
                    },
                    required: ['a', 'b', 'operation']
                }
            },
            {
                name: 'mcp_analyze',
                description: '📊 MCP 分析工具 - 分析文本内容',
                inputSchema: {
                    type: 'object',
                    properties: {
                        text: { type: 'string', description: '要分析的文本' }
                    },
                    required: ['text']
                }
            },
            {
                name: 'mcp_datetime',
                description: '📅 MCP 时间工具 - 获取日期时间',
                inputSchema: {
                    type: 'object',
                    properties: {
                        format: {
                            type: 'string',
                            enum: ['iso', 'unix', 'locale'],
                            description: '时间格式'
                        }
                    }
                }
            }
        ];
    }

    /**
     * 定义资源
     */
    defineResources() {
        return [
            {
                uri: 'poyo://config/plugin',
                name: '插件配置',
                description: 'MCP 插件的配置信息',
                mimeType: 'application/json'
            },
            {
                uri: 'poyo://info/server',
                name: '服务器信息',
                description: 'MCP 服务器的元信息',
                mimeType: 'application/json'
            },
            {
                uri: 'poyo://docs/readme',
                name: '使用文档',
                description: 'MCP 插件使用说明',
                mimeType: 'text/markdown'
            }
        ];
    }

    /**
     * 定义提示模板
     */
    definePrompts() {
        return [
            {
                name: 'mcp_greeting',
                description: '生成问候语模板',
                arguments: [
                    {
                        name: 'name',
                        description: '问候对象名称',
                        required: true
                    },
                    {
                        name: 'style',
                        description: '问候风格 (formal/casual)',
                        required: false
                    }
                ]
            },
            {
                name: 'mcp_analysis',
                description: '分析提示模板',
                arguments: [
                    {
                        name: 'topic',
                        description: '分析主题',
                        required: true
                    }
                ]
            }
        ];
    }

    /**
     * 定义请求处理器
     */
    defineRequestHandlers() {
        return {
            // 初始化
            'initialize': async (params) => {
                this.log('Received initialize request');
                return {
                    protocolVersion: '2024-11-05',
                    capabilities: {
                        tools: { supported: true, listChanged: true },
                        resources: { supported: true, subscribe: true, listChanged: true },
                        prompts: { supported: true },
                        sampling: { supported: true },
                        logging: { supported: true }
                    },
                    serverInfo: {
                        name: 'poyo-mcp-server',
                        version: '1.0.0'
                    }
                };
            },

            // 工具列表
            'tools/list': async () => ({
                tools: this.tools
            }),

            // 调用工具
            'tools/call': async (params) => {
                const { name, arguments: args } = params;
                return await this.executeTool(name, args);
            },

            // 资源列表
            'resources/list': async () => ({
                resources: this.resources
            }),

            // 读取资源
            'resources/read': async (params) => {
                const { uri } = params;
                return this.readResource(uri);
            },

            // 提示列表
            'prompts/list': async () => ({
                prompts: this.prompts
            }),

            // 获取提示
            'prompts/get': async (params) => {
                const { name, arguments: args } = params;
                return this.getPrompt(name, args);
            },

            // 日志级别设置
            'logging/setLevel': async (params) => {
                this.log(`Log level set to: ${params.level}`);
                return {};
            },

            // 根目录列表
            'roots/list': async () => ({
                roots: [
                    { uri: 'file:///workspace' }
                ]
            })
        };
    }

    /**
     * 执行工具
     */
    async executeTool(name, args) {
        this.log(`Executing tool: ${name}`);

        switch (name) {
            case 'mcp_hello':
                return {
                    content: [{
                        type: 'text',
                        text: `💚 你好, ${args.name}! 来自 MCP Poyo~\n\n我是通过 Model Context Protocol 连接的智能助手，很高兴为你服务!`
                    }]
                };

            case 'mcp_calculate':
                const { a, b, operation } = args;
                let result;
                switch (operation) {
                    case 'add': result = a + b; break;
                    case 'sub': result = a - b; break;
                    case 'mul': result = a * b; break;
                    case 'div': result = b !== 0 ? a / b : 'Error: Division by zero'; break;
                }
                return {
                    content: [{
                        type: 'text',
                        text: `🧮 计算结果:\n${a} ${operation} ${b} = ${result}`
                    }]
                };

            case 'mcp_analyze':
                const text = args.text;
                return {
                    content: [{
                        type: 'text',
                        text: `📊 文本分析结果:\n` +
                              `- 字符数: ${text.length}\n` +
                              `- 单词数: ${text.split(/\s+/).filter(w => w).length}\n` +
                              `- 行数: ${text.split('\n').length}\n` +
                              `- 包含中文: ${/[\u4e00-\u9fff]/.test(text)}\n` +
                              `- 分析时间: ${new Date().toISOString()}`
                    }]
                };

            case 'mcp_datetime':
                const format = args.format || 'iso';
                let timeStr;
                switch (format) {
                    case 'unix': timeStr = Math.floor(Date.now() / 1000).toString(); break;
                    case 'locale': timeStr = new Date().toLocaleString(); break;
                    default: timeStr = new Date().toISOString();
                }
                return {
                    content: [{
                        type: 'text',
                        text: `📅 当前时间 (${format}): ${timeStr}`
                    }]
                };

            default:
                return {
                    content: [{
                        type: 'text',
                        text: `未知工具: ${name}`
                    }],
                    isError: true
                };
        }
    }

    /**
     * 读取资源
     */
    readResource(uri) {
        this.log(`Reading resource: ${uri}`);

        const contents = {
            'poyo://config/plugin': {
                content: JSON.stringify({
                    name: 'MCP Server Example',
                    version: '1.0.0',
                    enabled: true,
                    tools: this.tools.length,
                    resources: this.resources.length,
                    prompts: this.prompts.length
                }, null, 2)
            },
            'poyo://info/server': {
                content: JSON.stringify({
                    name: 'poyo-mcp-server',
                    version: '1.0.0',
                    protocolVersion: '2024-11-05',
                    capabilities: ['tools', 'resources', 'prompts', 'sampling', 'logging'],
                    startTime: new Date().toISOString()
                }, null, 2)
            },
            'poyo://docs/readme': {
                content: `# MCP Server Example

## 🔌 这是一个 MCP (Model Context Protocol) 服务器示例

### 功能特性

- **Tools**: 提供 4 个可调用工具
  - mcp_hello: 问候工具
  - mcp_calculate: 计算工具
  - mcp_analyze: 文本分析工具
  - mcp_datetime: 时间工具

- **Resources**: 提供 3 个资源
  - 插件配置
  - 服务器信息
  - 使用文档

- **Prompts**: 提供 2 个提示模板
  - mcp_greeting: 问候语模板
  - mcp_analysis: 分析提示模板

### 使用方法

通过 MCP 客户端连接此服务器，即可使用上述所有功能。
`
            }
        };

        if (contents[uri]) {
            return {
                contents: [{
                    uri,
                    mimeType: uri.endsWith('.json') ? 'application/json' : 'text/markdown',
                    text: contents[uri].content
                }]
            };
        }

        return {
            contents: [],
            error: `Resource not found: ${uri}`
        };
    }

    /**
     * 获取提示
     */
    getPrompt(name, args) {
        this.log(`Getting prompt: ${name}`);

        switch (name) {
            case 'mcp_greeting':
                const style = args.style || 'casual';
                return {
                    messages: [{
                        role: 'user',
                        content: {
                            type: 'text',
                            text: style === 'formal'
                                ? `请用正式的方式问候 ${args.name}，表达诚挚的欢迎之意。`
                                : `用轻松友好的方式跟 ${args.name} 打个招呼吧!`
                        }
                    }]
                };

            case 'mcp_analysis':
                return {
                    messages: [{
                        role: 'user',
                        content: {
                            type: 'text',
                            text: `请对以下主题进行深入分析: ${args.topic}\n\n分析要点:\n1. 核心概念\n2. 主要特点\n3. 应用场景\n4. 未来展望`
                        }
                    }]
                };

            default:
                return {
                    messages: [],
                    error: `Prompt not found: ${name}`
                };
        }
    }

    /**
     * 日志
     */
    log(message, level = 'info') {
        console.error(`[MCP][${level.toUpperCase()}] ${message}`);
    }

    /**
     * 发送响应
     */
    sendResponse(id, result) {
        const response = {
            jsonrpc: '2.0',
            id,
            result
        };
        console.log(JSON.stringify(response));
    }

    /**
     * 发送错误
     */
    sendError(id, code, message) {
        const response = {
            jsonrpc: '2.0',
            id,
            error: { code, message }
        };
        console.log(JSON.stringify(response));
    }

    /**
     * 发送通知
     */
    sendNotification(method, params) {
        const notification = {
            jsonrpc: '2.0',
            method,
            params
        };
        console.log(JSON.stringify(notification));
    }

    /**
     * 处理请求
     */
    async handleRequest(request) {
        const { id, method, params } = request;

        this.log(`Handling request: ${method}`);

        if (this.requestHandlers[method]) {
            try {
                const result = await this.requestHandlers[method](params || {});
                this.sendResponse(id, result);
            } catch (e) {
                this.sendError(id, -32000, e.message);
            }
        } else {
            this.sendError(id, -32601, `Method not found: ${method}`);
        }
    }

    /**
     * 启动服务器
     */
    start() {
        this.log('Starting MCP server...');

        this.rl.on('line', (line) => {
            try {
                const request = JSON.parse(line);
                this.handleRequest(request);
            } catch (e) {
                this.log(`Parse error: ${e.message}`, 'error');
            }
        });

        this.rl.on('close', () => {
            this.log('Server shutting down');
        });
    }
}

// 启动服务器
const server = new MCPServer();
server.start();
