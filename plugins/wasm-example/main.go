//go:build wasi || wasm
// +build wasi wasm

/**
 * ⚡ Poyo WASM Plugin Example
 * 展示 WebAssembly 插件的所有能力
 *
 * 编译命令:
 * GOOS=wasip1 GOARCH=wasm go build -o main.wasm main.go
 *
 * 或使用 TinyGo:
 * tinygo build -o main.wasm -target wasi main.go
 */

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"unsafe"
)

// ═══════════════════════════════════════════════════════
// 🔧 Types
// ═══════════════════════════════════════════════════════

// Input represents the input passed to the WASM plugin
type Input struct {
	Method string                 `json:"method"`
	Args   map[string]interface{} `json:"args"`
	Env    Environment            `json:"env"`
}

// Environment represents the plugin environment
type Environment struct {
	PluginID      string                 `json:"pluginId"`
	PluginName    string                 `json:"pluginName"`
	PluginVersion string                 `json:"pluginVersion"`
	DreamLand     string                 `json:"dreamLand"`
	Config        map[string]interface{} `json:"config"`
}

// Output represents the output from the WASM plugin
type Output struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
	Logs    []string    `json:"logs,omitempty"`
}

// ═══════════════════════════════════════════════════════
// 🔧 WASM Export Functions
// ═══════════════════════════════════════════════════════

// 为 WASM 导出的内存操作函数
var inputBuffer []byte
var outputBuffer []byte

//export allocate
func allocate(size int) unsafe.Pointer {
	buf := make([]byte, size)
	inputBuffer = buf
	return unsafe.Pointer(&buf[0])
}

//export deallocate
func deallocate(ptr unsafe.Pointer, size int) {
	// Go 的 GC 会处理
}

//export get_output_ptr
func get_output_ptr() unsafe.Pointer {
	if len(outputBuffer) > 0 {
		return unsafe.Pointer(&outputBuffer[0])
	}
	return unsafe.Pointer(nil)
}

//export get_output_len
func get_output_len() int {
	return len(outputBuffer)
}

//export process
func process(inputLen int) int {
	// 解析输入
	var input Input
	if err := json.Unmarshal(inputBuffer[:inputLen], &input); err != nil {
		outputBuffer = marshalOutput(Output{
			Success: false,
			Error:   fmt.Sprintf("解析输入失败: %v", err),
		})
		return len(outputBuffer)
	}

	// 处理请求
	result := handleRequest(input)

	// 序列化输出
	outputBuffer = marshalOutput(result)
	return len(outputBuffer)
}

// ═══════════════════════════════════════════════════════
// 🎯 Tool Implementations
// ═══════════════════════════════════════════════════════

func toolWasmHello(args map[string]interface{}) Output {
	name := getString(args, "name", "World")

	return Output{
		Success: true,
		Result: map[string]interface{}{
			"greeting":   fmt.Sprintf("⚡ 你好, %s! 来自 WebAssembly~", name),
			"runtime":    "WebAssembly (WASI)",
			"timestamp":  time.Now().Format(time.RFC3339),
			"pluginType": "wasm",
		},
		Logs: []string{fmt.Sprintf("[INFO] Greeted %s", name)},
	}
}

func toolWasmCompute(args map[string]interface{}) Output {
	operation := getString(args, "operation", "fibonacci")
	n := getInt(args, "n", 10)

	var result interface{}
	var logs []string

	start := time.Now()

	switch operation {
	case "fibonacci":
		result = fibonacci(n)
		logs = append(logs, fmt.Sprintf("[INFO] Computed fibonacci(%d)", n))

	case "factorial":
		result = factorial(n)
		logs = append(logs, fmt.Sprintf("[INFO] Computed factorial(%d)", n))

	case "primes":
		result = sieveOfEratosthenes(n)
		logs = append(logs, fmt.Sprintf("[INFO] Found primes up to %d", n))

	case "collatz":
		steps := collatzSteps(n)
		result = map[string]interface{}{
			"n":     n,
			"steps": steps,
		}
		logs = append(logs, fmt.Sprintf("[INFO] Collatz steps for %d: %d", n, steps))

	default:
		return Output{
			Success: false,
			Error:   fmt.Sprintf("未知操作: %s", operation),
		}
	}

	elapsed := time.Since(start)
	logs = append(logs, fmt.Sprintf("[INFO] Computation took %v", elapsed))

	return Output{
		Success: true,
		Result: map[string]interface{}{
			"operation": operation,
			"input":     n,
			"result":    result,
			"duration":  elapsed.String(),
		},
		Logs: logs,
	}
}

func toolWasmMemory(args map[string]interface{}) Output {
	operation := getString(args, "operation", "info")

	var result interface{}

	switch operation {
	case "info":
		var m runtimeStats
		result = map[string]interface{}{
			"inputBufferSize":  len(inputBuffer),
			"outputBufferSize": len(outputBuffer),
			"stats":            m,
		}

	case "allocate":
		size := getInt(args, "size", 1024)
		_ = make([]byte, size) // 分配内存
		result = map[string]interface{}{
			"allocated": size,
			"message":   fmt.Sprintf("分配了 %d 字节", size),
		}

	default:
		return Output{
			Success: false,
			Error:   fmt.Sprintf("未知操作: %s", operation),
		}
	}

	return Output{
		Success: true,
		Result:  result,
		Logs:    []string{fmt.Sprintf("[INFO] Memory operation: %s", operation)},
	}
}

func toolWasmString(args map[string]interface{}) Output {
	text := getString(args, "text", "")
	operation := getString(args, "operation", "reverse")

	var result string

	switch operation {
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		result = string(runes)

	case "upper":
		result = stringsToUpper(text)

	case "lower":
		result = stringsToLower(text)

	case "length":
		return Output{
			Success: true,
			Result: map[string]interface{}{
				"text":       text,
				"length":     len(text),
				"runeCount":  len([]rune(text)),
				"byteCount":  len(text),
				"operation":  operation,
			},
		}

	default:
		return Output{
			Success: false,
			Error:   fmt.Sprintf("未知操作: %s", operation),
		}
	}

	return Output{
		Success: true,
		Result: map[string]interface{}{
			"original":  text,
			"result":    result,
			"operation": operation,
		},
	}
}

// ═══════════════════════════════════════════════════════
// 🪝 Hook Implementations
// ═══════════════════════════════════════════════════════

func hookPreToolUse(input Input) Output {
	tool := getString(input.Args, "tool", "unknown")

	return Output{
		Success: true,
		Result: map[string]interface{}{
			"blocked": false,
			"message": fmt.Sprintf("WASM 插件允许执行 %s", tool),
			"source":  "wasm-plugin",
		},
		Logs: []string{fmt.Sprintf("[DEBUG] PreToolUse: %s", tool)},
	}
}

func hookPostToolUse(input Input) Output {
	tool := getString(input.Args, "tool", "unknown")
	success := getBool(input.Args, "success", false)

	var logs []string
	if success {
		logs = append(logs, fmt.Sprintf("[INFO] Tool %s executed successfully", tool))
	} else {
		logs = append(logs, fmt.Sprintf("[WARN] Tool %s failed", tool))
	}

	return Output{
		Success: true,
		Result: map[string]interface{}{
			"logged":  true,
			"tool":    tool,
			"success": success,
			"source":  "wasm-plugin",
		},
		Logs: logs,
	}
}

// ═══════════════════════════════════════════════════════
// 🚀 Command Implementations
// ═══════════════════════════════════════════════════════

func commandWasmDemo(input Input) Output {
	return Output{
		Success: true,
		Result: map[string]interface{}{
			"message": "⚡ WebAssembly 插件演示完成!",
			"environment": map[string]interface{}{
				"pluginId":      input.Env.PluginID,
				"pluginName":    input.Env.PluginName,
				"pluginVersion": input.Env.PluginVersion,
				"dreamLand":     input.Env.DreamLand,
			},
			"availableTools": []string{
				"wasm_hello",
				"wasm_compute",
				"wasm_memory",
				"wasm_string",
			},
			"availableHooks": []string{
				"PreToolUse",
				"PostToolUse",
			},
			"runtime": "WebAssembly (WASI)",
		},
		Logs: []string{
			"[INFO] Demo command executed",
			"[INFO] WASM plugin is working!",
		},
	}
}

// ═══════════════════════════════════════════════════════
// 📋 Main Dispatcher
// ═══════════════════════════════════════════════════════

func handleRequest(input Input) Output {
	method := input.Method

	// 方法路由
	switch method {
	// Tools
	case "wasm_hello", "tool_wasm_hello":
		return toolWasmHello(input.Args)
	case "wasm_compute", "tool_wasm_compute":
		return toolWasmCompute(input.Args)
	case "wasm_memory", "tool_wasm_memory":
		return toolWasmMemory(input.Args)
	case "wasm_string", "tool_wasm_string":
		return toolWasmString(input.Args)

	// Hooks
	case "PreToolUse":
		return hookPreToolUse(input)
	case "PostToolUse":
		return hookPostToolUse(input)

	// Commands
	case "wasm_demo", "command_wasm_demo":
		return commandWasmDemo(input)

	default:
		return Output{
			Success: false,
			Error:   fmt.Sprintf("未知方法: %s", method),
			Logs:    []string{fmt.Sprintf("[ERROR] Unknown method: %s", method)},
		}
	}
}

// ═══════════════════════════════════════════════════════
// 🔧 Helper Functions
// ═══════════════════════════════════════════════════════

func marshalOutput(output Output) []byte {
	data, _ := json.Marshal(output)
	return data
}

func getString(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getInt(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return defaultVal
}

func getBool(m map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// ═══════════════════════════════════════════════════════
// 🧮 Computation Functions
// ═══════════════════════════════════════════════════════

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

func factorial(n int) int {
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}

func sieveOfEratosthenes(n int) []int {
	if n < 2 {
		return []int{}
	}
	sieve := make([]bool, n+1)
	for i := 2; i <= n; i++ {
		sieve[i] = true
	}
	for i := 2; i*i <= n; i++ {
		if sieve[i] {
			for j := i * i; j <= n; j += i {
				sieve[j] = false
			}
		}
	}
	var primes []int
	for i := 2; i <= n; i++ {
		if sieve[i] {
			primes = append(primes, i)
		}
	}
	return primes
}

func collatzSteps(n int) int {
	steps := 0
	for n != 1 {
		if n%2 == 0 {
			n = n / 2
		} else {
			n = 3*n + 1
		}
		steps++
	}
	return steps
}

func stringsToUpper(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'a' && r <= 'z' {
			result[i] = r - 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func stringsToLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// runtimeStats 模拟运行时统计
type runtimeStats struct {
	Goroutines int `json:"goroutines"`
}

// ═══════════════════════════════════════════════════════
// 🚀 Main
// ═══════════════════════════════════════════════════════

func main() {
	// WASI 模式下，等待调用导出的函数
	// 主函数在这里只是占位
	fmt.Fprintln(os.Stderr, "[WASM] Plugin loaded and ready")
}
