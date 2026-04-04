#!/usr/bin/env node
/**
 * 🔍 OpenClaw Code Analyzer Plugin
 * 代码分析工具 - OpenClaw 格式示例
 *
 * OpenClaw 格式特点:
 * - 使用 openclaw.json 作为 manifest
 * - runtime 指定运行时环境
 * - provides 定义提供的工具、命令、钩子和路由
 * - requires 定义依赖和权限
 */

const fs = require('fs');
const path = require('path');

// ═══════════════════════════════════════════════════════
// 🔧 Poyo API Helper
// ═══════════════════════════════════════════════════════

const Poyo = {
    getInput() {
        try {
            const input = fs.readFileSync(0, 'utf-8');
            return JSON.parse(input);
        } catch (e) {
            return {};
        }
    },

    output(result, success = true, error = null) {
        const output = { success, result, timestamp: new Date().toISOString() };
        if (error) output.error = error;
        console.log(JSON.stringify(output, null, 2));
    },

    getEnv() {
        return {
            pluginId: process.env.POYO_PLUGIN_ID || 'code-analyzer',
            dreamLand: process.env.POYO_DREAM_LAND || process.cwd(),
            config: JSON.parse(process.env.POYO_PLUGIN_CONFIG || '{}')
        };
    },

    log(message, level = 'info') {
        console.error(`[${level.toUpperCase()}] [CodeAnalyzer] ${message}`);
    }
};

// ═══════════════════════════════════════════════════════
// 🎯 Tool Implementations
// ═══════════════════════════════════════════════════════

/**
 * 分析代码质量
 */
function analyzeCode(args) {
    const targetPath = args.path;
    const rules = args.rules || ['best-practices', 'security', 'performance'];
    const format = args.format || 'json';
    const env = Poyo.getEnv();

    // 解析路径
    const fullPath = path.isAbsolute(targetPath)
        ? targetPath
        : path.join(env.dreamLand, targetPath);

    Poyo.log(`Analyzing: ${fullPath}`);

    // 检查路径是否存在
    if (!fs.existsSync(fullPath)) {
        return { error: `Path not found: ${fullPath}` };
    }

    const issues = [];
    const metrics = {
        files: 0,
        lines: 0,
        functions: 0,
        classes: 0
    };

    // 分析文件或目录
    const analyzeFile = (filePath) => {
        try {
            const stat = fs.statSync(filePath);
            if (stat.isDirectory()) {
                const entries = fs.readdirSync(filePath);
                for (const entry of entries) {
                    // 跳过排除的目录
                    if (env.config.excludePatterns?.includes(entry)) continue;
                    analyzeFile(path.join(filePath, entry));
                }
                return;
            }

            // 只分析代码文件
            const ext = path.extname(filePath);
            if (!['.js', '.ts', '.py', '.go', '.java', '.lua'].includes(ext)) return;

            metrics.files++;
            const content = fs.readFileSync(filePath, 'utf-8');
            const lines = content.split('\n');
            metrics.lines += lines.length;

            // 简单的代码分析
            for (let i = 0; i < lines.length; i++) {
                const line = lines[i];
                const lineNum = i + 1;

                // 检测函数
                if (/function\s+\w+|def\s+\w+|func\s+\w+/.test(line)) {
                    metrics.functions++;
                }

                // 检测类
                if (/class\s+\w+/.test(line)) {
                    metrics.classes++;
                }

                // 安全规则
                if (rules.includes('security')) {
                    // 检测硬编码密码
                    if (/password\s*=\s*['"]/.test(line) && !/password\s*=\s*process\.env/.test(line)) {
                        issues.push({
                            file: filePath,
                            line: lineNum,
                            rule: 'security:hardcoded-password',
                            severity: 'warning',
                            message: 'Potential hardcoded password detected'
                        });
                    }

                    // 检测 eval 使用
                    if (/\beval\s*\(/.test(line)) {
                        issues.push({
                            file: filePath,
                            line: lineNum,
                            rule: 'security:eval-usage',
                            severity: 'error',
                            message: 'Avoid using eval() - security risk'
                        });
                    }
                }

                // 最佳实践
                if (rules.includes('best-practices')) {
                    // 检测长行
                    if (line.length > 120) {
                        issues.push({
                            file: filePath,
                            line: lineNum,
                            rule: 'best-practices:line-length',
                            severity: 'info',
                            message: `Line exceeds 120 characters (${line.length} chars)`
                        });
                    }

                    // 检测 TODO
                    if (/TODO|FIXME|HACK|XXX/.test(line)) {
                        issues.push({
                            file: filePath,
                            line: lineNum,
                            rule: 'best-practices:todo',
                            severity: 'info',
                            message: 'TODO/FIXME comment found'
                        });
                    }
                }
            }
        } catch (e) {
            Poyo.log(`Error analyzing ${filePath}: ${e.message}`, 'warn');
        }
    };

    analyzeFile(fullPath);

    // 生成摘要
    const summary = `Analyzed ${metrics.files} files, ${metrics.lines} lines. ` +
                   `Found ${issues.filter(i => i.severity === 'error').length} errors, ` +
                   `${issues.filter(i => i.severity === 'warning').length} warnings.`;

    Poyo.log(summary);

    const result = {
        path: targetPath,
        rules,
        metrics,
        issues,
        summary
    };

    // 根据格式返回
    if (format === 'markdown') {
        return {
            format: 'markdown',
            content: `# Code Analysis Report\n\n${summary}\n\n## Metrics\n- Files: ${metrics.files}\n- Lines: ${metrics.lines}\n- Functions: ${metrics.functions}\n- Classes: ${metrics.classes}\n\n## Issues\n${issues.map(i => `- [${i.severity}] ${i.file}:${i.line} - ${i.message}`).join('\n')}`
        };
    }

    return result;
}

/**
 * 检测设计模式
 */
function detectPatterns(args) {
    const targetPath = args.path;
    const patterns = args.patterns || ['singleton', 'factory', 'observer', 'decorator'];
    const env = Poyo.getEnv();

    const fullPath = path.isAbsolute(targetPath)
        ? targetPath
        : path.join(env.dreamLand, targetPath);

    Poyo.log(`Detecting patterns in: ${fullPath}`);

    if (!fs.existsSync(fullPath)) {
        return { error: `Path not found: ${fullPath}` };
    }

    const detected = [];
    const content = fs.readFileSync(fullPath, 'utf-8');

    // 简单的模式检测
    if (patterns.includes('singleton')) {
        if (/getInstance\s*\(|new\s+\w+\(\s*\)\s*;?\s*return\s+instance/.test(content)) {
            detected.push({ pattern: 'singleton', confidence: 0.8 });
        }
    }

    if (patterns.includes('factory')) {
        if (/create\w+\s*\(|factory\s*\(/i.test(content)) {
            detected.push({ pattern: 'factory', confidence: 0.7 });
        }
    }

    if (patterns.includes('observer')) {
        if (/subscribe|notify|observer|listener/i.test(content)) {
            detected.push({ pattern: 'observer', confidence: 0.6 });
        }
    }

    if (patterns.includes('decorator')) {
        if /@|\bwrapper\b|decorate/i.test(content)) {
            detected.push({ pattern: 'decorator', confidence: 0.5 });
        }
    }

    return {
        path: targetPath,
        detected,
        patternsChecked: patterns
    };
}

/**
 * 计算代码复杂度
 */
function calculateComplexity(args) {
    const targetPath = args.path;
    const type = args.type || 'cyclomatic';
    const env = Poyo.getEnv();

    const fullPath = path.isAbsolute(targetPath)
        ? targetPath
        : path.join(env.dreamLand, targetPath);

    Poyo.log(`Calculating ${type} complexity for: ${fullPath}`);

    if (!fs.existsSync(fullPath)) {
        return { error: `Path not found: ${fullPath}` };
    }

    const content = fs.readFileSync(fullPath, 'utf-8');
    const lines = content.split('\n');

    let complexity = 1; // 基础复杂度

    if (type === 'cyclomatic') {
        // 圈复杂度：计算分支数量
        const branchPatterns = /\b(if|else|for|while|case|catch|&&|\|\|)\b/g;
        const matches = content.match(branchPatterns);
        complexity += matches ? matches.length : 0;
    } else if (type === 'cognitive') {
        // 认知复杂度：更复杂的度量
        // 简化实现
        const indentLevels = [];
        let maxIndent = 0;
        for (const line of lines) {
            const indent = line.search(/\S/);
            if (indent > 0) {
                maxIndent = Math.max(maxIndent, Math.floor(indent / 4));
            }
        }
        complexity = maxIndent * 2;
    }

    const rating = complexity <= 5 ? 'simple' : complexity <= 10 ? 'moderate' : 'complex';

    return {
        path: targetPath,
        type,
        value: complexity,
        rating,
        recommendation: complexity > 10 ? 'Consider refactoring to reduce complexity' : 'Complexity is acceptable'
    };
}

// ═══════════════════════════════════════════════════════
// 🪝 Hook Implementations
// ═══════════════════════════════════════════════════════

function beforeToolUse(input) {
    const tool = input.tool || 'unknown';
    Poyo.log(`Tool about to execute: ${tool}`, 'debug');
    return { blocked: false, source: 'code-analyzer' };
}

function afterToolUse(input) {
    const tool = input.tool || 'unknown';
    const success = input.success;
    Poyo.log(`Tool ${tool} ${success ? 'succeeded' : 'failed'}`);
    return { logged: true };
}

// ═══════════════════════════════════════════════════════
// 🚀 Command Implementations
// ═══════════════════════════════════════════════════════

function analyzeProject(args) {
    const env = Poyo.getEnv();
    return analyzeCode({ path: env.dreamLand, ...args });
}

// ═══════════════════════════════════════════════════════
// 🌐 HTTP Route Handler
// ═══════════════════════════════════════════════════════

function httpAnalyze(params) {
    return analyzeCode({ path: params.path });
}

// ═══════════════════════════════════════════════════════
// 📋 Main Dispatcher
// ═══════════════════════════════════════════════════════

function main() {
    const input = Poyo.getInput();
    const method = input.method || '';
    const args = input.args || {};

    // OpenClaw 方法路由 (转换格式)
    const methodMap = {
        // Tools
        'analyze_code': analyzeCode,
        'detect_patterns': detectPatterns,
        'calculate_complexity': calculateComplexity,

        // Hooks (OpenClaw 事件格式)
        'tool.use.before': beforeToolUse,
        'tool.use.after': afterToolUse,

        // Commands
        'analyze-project': analyzeProject,

        // Routes
        'httpAnalyze': httpAnalyze
    };

    if (!methodMap[method]) {
        Poyo.output(null, false, `Unknown method: ${method}`);
        return;
    }

    try {
        const fn = methodMap[method];
        // Hooks 接收完整 input
        const result = method.includes('tool.use')
            ? fn(input)
            : fn(args);
        Poyo.output(result);
    } catch (e) {
        Poyo.output(null, false, e.message);
    }
}

main();
