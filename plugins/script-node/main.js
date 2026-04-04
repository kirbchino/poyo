#!/usr/bin/env node
/**
 * 📦 Poyo Node.js Script Plugin Example
 * 展示 Node.js 脚本插件的所有能力
 *
 * 环境变量说明:
 * - POYO_API_VERSION: API 版本
 * - POYO_PLUGIN_ID: 插件 ID
 * - POYO_PLUGIN_NAME: 插件名称
 * - POYO_PLUGIN_VERSION: 插件版本
 * - POYO_PLUGIN_PATH: 插件路径
 * - POYO_DREAM_LAND: 梦之国（工作目录）
 * - POYO_PLUGIN_CONFIG: 插件配置 (JSON)
 */

const fs = require('fs');
const path = require('path');

// ═══════════════════════════════════════════════════════
// 🔧 Poyo API Helper
// ═══════════════════════════════════════════════════════

const Poyo = {
    /**
     * 从 stdin 获取输入
     */
    getInput() {
        try {
            const input = fs.readFileSync(0, 'utf-8');
            return JSON.parse(input);
        } catch (e) {
            return {};
        }
    },

    /**
     * 输出结果到 stdout
     */
    output(result, success = true, error = null) {
        const output = {
            success,
            result,
            timestamp: new Date().toISOString()
        };
        if (error) {
            output.error = error;
        }
        console.log(JSON.stringify(output, null, 2));
    },

    /**
     * 获取插件环境
     */
    getEnv() {
        let config = {};
        try {
            config = JSON.parse(process.env.POYO_PLUGIN_CONFIG || '{}');
        } catch (e) {}

        return {
            apiVersion: process.env.POYO_API_VERSION || '',
            pluginId: process.env.POYO_PLUGIN_ID || '',
            pluginName: process.env.POYO_PLUGIN_NAME || '',
            pluginVersion: process.env.POYO_PLUGIN_VERSION || '',
            pluginPath: process.env.POYO_PLUGIN_PATH || '',
            dreamLand: process.env.POYO_DREAM_LAND || '',
            config
        };
    },

    /**
     * 日志输出到 stderr
     */
    log(message, level = 'info') {
        console.error(`[${level.toUpperCase()}] ${message}`);
    },

    /**
     * Poyo 说话！
     */
    say(message) {
        console.error(`💚 Poyo 说: ${message}`);
    },

    /**
     * 舞蹈！
     */
    dance() {
        console.error('💃 Poyo 跳舞中~ 💚🌀💚');
    }
};

// ═══════════════════════════════════════════════════════
// 🎯 Tool Implementations
// ═══════════════════════════════════════════════════════

/**
 * JSON 处理工具
 */
function toolNodeJson(args) {
    const operation = args.operation || 'stringify';
    const data = args.data;

    try {
        let result;
        switch (operation) {
            case 'stringify':
                result = JSON.stringify(data, null, 2);
                break;
            case 'parse':
                result = JSON.parse(data);
                break;
            case 'keys':
                result = Object.keys(data);
                break;
            case 'values':
                result = Object.values(data);
                break;
            case 'entries':
                result = Object.entries(data);
                break;
            case 'merge':
                result = { ...data, ...args.mergeWith };
                break;
            default:
                return { error: `未知操作: ${operation}` };
        }

        Poyo.log(`JSON 操作完成: ${operation}`);
        return { operation, result };
    } catch (e) {
        return { error: `JSON 处理错误: ${e.message}` };
    }
}

/**
 * 文件处理工具
 */
function toolNodeFile(args) {
    const operation = args.operation || 'read';
    const filePath = args.path;
    const env = Poyo.getEnv();

    // 相对路径处理
    const fullPath = path.isAbsolute(filePath)
        ? filePath
        : path.join(env.dreamLand, filePath);

    try {
        let result;
        switch (operation) {
            case 'read':
                result = fs.readFileSync(fullPath, 'utf-8');
                break;
            case 'exists':
                result = fs.existsSync(fullPath);
                break;
            case 'stats':
                const stats = fs.statSync(fullPath);
                result = {
                    size: stats.size,
                    isFile: stats.isFile(),
                    isDirectory: stats.isDirectory(),
                    modified: stats.mtime
                };
                break;
            case 'list':
                result = fs.readdirSync(fullPath);
                break;
            default:
                return { error: `未知操作: ${operation}` };
        }

        Poyo.log(`文件操作完成: ${operation} on ${fullPath}`);
        return { operation, path: fullPath, result };
    } catch (e) {
        return { error: `文件操作错误: ${e.message}` };
    }
}

/**
 * 时间处理工具
 */
function toolNodeTime(args) {
    const operation = args.operation || 'now';
    const format = args.format || 'iso';

    const now = new Date();

    try {
        let result;
        switch (operation) {
            case 'now':
                result = now.toISOString();
                break;
            case 'timestamp':
                result = now.getTime();
                break;
            case 'format':
                // 简单格式化
                const options = {
                    'iso': () => now.toISOString(),
                    'date': () => now.toDateString(),
                    'time': () => now.toTimeString(),
                    'locale': () => now.toLocaleString(),
                    'unix': () => Math.floor(now.getTime() / 1000)
                };
                result = options[format] ? options[format]() : now.toISOString();
                break;
            case 'add':
                const ms = args.milliseconds || 0;
                result = new Date(now.getTime() + ms).toISOString();
                break;
            default:
                return { error: `未知操作: ${operation}` };
        }

        Poyo.log(`时间操作完成: ${operation}`);
        return { operation, result };
    } catch (e) {
        return { error: `时间操作错误: ${e.message}` };
    }
}

/**
 * 数据处理工具
 */
function toolNodeData(args) {
    const operation = args.operation || 'transform';
    const data = args.data;

    try {
        let result;
        switch (operation) {
            case 'transform':
                // 转换数据
                const transform = args.transform || 'identity';
                if (transform === 'uppercase' && typeof data === 'string') {
                    result = data.toUpperCase();
                } else if (transform === 'lowercase' && typeof data === 'string') {
                    result = data.toLowerCase();
                } else if (transform === 'reverse' && Array.isArray(data)) {
                    result = data.reverse();
                } else if (transform === 'sort' && Array.isArray(data)) {
                    result = [...data].sort();
                } else if (transform === 'unique' && Array.isArray(data)) {
                    result = [...new Set(data)];
                } else {
                    result = data;
                }
                break;
            case 'filter':
                // 过滤数组
                const predicate = args.predicate;
                if (Array.isArray(data) && predicate) {
                    result = data.filter(item => {
                        if (predicate.type === 'equals') {
                            return item[predicate.key] === predicate.value;
                        }
                        return true;
                    });
                } else {
                    result = data;
                }
                break;
            case 'map':
                // 映射数组
                const mapper = args.mapper;
                if (Array.isArray(data) && mapper) {
                    result = data.map(item => item[mapper.key]);
                } else {
                    result = data;
                }
                break;
            case 'reduce':
                // 归约数组
                const reducer = args.reducer;
                if (Array.isArray(data) && reducer === 'sum') {
                    result = data.reduce((a, b) => a + b, 0);
                } else if (Array.isArray(data) && reducer === 'count') {
                    result = data.length;
                } else {
                    result = data;
                }
                break;
            default:
                return { error: `未知操作: ${operation}` };
        }

        Poyo.log(`数据操作完成: ${operation}`);
        return { operation, result };
    } catch (e) {
        return { error: `数据操作错误: ${e.message}` };
    }
}

// ═══════════════════════════════════════════════════════
// 🪝 Hook Implementations
// ═══════════════════════════════════════════════════════

function hookPostToolUse(input) {
    const tool = input.tool || 'unknown';
    const success = input.success;

    if (success) {
        Poyo.log(`工具 ${tool} 执行成功 (Node.js 监控)`);
    } else {
        Poyo.log(`工具 ${tool} 执行失败 (Node.js 监控)`, 'warn');
    }

    return {
        logged: true,
        monitoredBy: 'node-plugin'
    };
}

// ═══════════════════════════════════════════════════════
// 🚀 Command Implementations
// ═══════════════════════════════════════════════════════

function commandNodeDemo() {
    const env = Poyo.getEnv();

    Poyo.say('欢迎来到 Node.js 插件演示!');
    Poyo.dance();

    return {
        message: '📦 Node.js 插件演示完成!',
        environment: {
            pluginId: env.pluginId,
            pluginName: env.pluginName,
            dreamLand: env.dreamLand
        },
        availableTools: ['node_json', 'node_file', 'node_time', 'node_data'],
        nodeVersion: process.version,
        platform: process.platform
    };
}

// ═══════════════════════════════════════════════════════
// 📋 Main Dispatcher
// ═══════════════════════════════════════════════════════

function main() {
    // 获取输入
    const input = Poyo.getInput();
    const method = input.method || '';
    const args = input.args || {};

    Poyo.log(`收到方法调用: ${method}`, 'debug');

    // 方法路由
    const methods = {
        // Tools
        'node_json': toolNodeJson,
        'tool_node_json': toolNodeJson,
        'node_file': toolNodeFile,
        'tool_node_file': toolNodeFile,
        'node_time': toolNodeTime,
        'tool_node_time': toolNodeTime,
        'node_data': toolNodeData,
        'tool_node_data': toolNodeData,

        // Hooks
        'PostToolUse': hookPostToolUse,

        // Commands
        'node_demo': commandNodeDemo,
        'command_node_demo': commandNodeDemo,
    };

    if (!methods[method]) {
        Poyo.output(null, false, `未知方法: ${method}`);
        return;
    }

    try {
        // Hooks 接收完整 input，其他接收 args
        const result = method.startsWith('Pre') || method.startsWith('Post')
            ? methods[method](input)
            : methods[method](args);
        Poyo.output(result);
    } catch (e) {
        Poyo.output(null, false, e.message);
    }
}

// 运行
main();
