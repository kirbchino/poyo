// Package plugin provides WebAssembly plugin support
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WASMPluginHandler handles WebAssembly-based plugins
type WASMPluginHandler struct {
	mu         sync.RWMutex
	runtime    wazero.Runtime
	modules    map[string]api.Module // Plugin ID -> Module
	workingDir string
}

// NewWASMPluginHandler creates a new WASM plugin handler
func NewWASMPluginHandler(workingDir string) *WASMPluginHandler {
	// Create a new WebAssembly runtime
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)

	return &WASMPluginHandler{
		runtime:    rt,
		modules:    make(map[string]api.Module),
		workingDir: workingDir,
	}
}

// Load loads a WASM plugin
func (h *WASMPluginHandler) Load(ctx context.Context, plugin *Plugin) error {
	if plugin.Main == "" {
		return fmt.Errorf("no main WASM file specified")
	}

	// Read WASM file
	wasmPath := plugin.Main
	if !isAbs(wasmPath) {
		wasmPath = joinPath(plugin.Path, plugin.Main)
	}

	wasmBytes, err := readFile(wasmPath)
	if err != nil {
		return fmt.Errorf("read WASM file: %w", err)
	}

	// Compile and instantiate the module
	module, err := h.runtime.Instantiate(ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("instantiate WASM module: %w", err)
	}

	h.mu.Lock()
	h.modules[plugin.ID] = module
	h.mu.Unlock()

	return nil
}

// Unload unloads a WASM plugin
func (h *WASMPluginHandler) Unload(ctx context.Context, plugin *Plugin) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	module, ok := h.modules[plugin.ID]
	if !ok {
		return nil
	}

	if err := module.Close(ctx); err != nil {
		return fmt.Errorf("close WASM module: %w", err)
	}

	delete(h.modules, plugin.ID)
	return nil
}

// Execute executes a WASM plugin method
func (h *WASMPluginHandler) Execute(ctx context.Context, plugin *Plugin, method string, input interface{}) (interface{}, error) {
	h.mu.RLock()
	module, ok := h.modules[plugin.ID]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %s not loaded", plugin.ID)
	}

	// Build input
	inputData := map[string]interface{}{
		"method": method,
		"args":   input,
		"env": map[string]interface{}{
			"pluginId":      plugin.ID,
			"pluginName":    plugin.Name,
			"pluginVersion": plugin.Version,
			"dreamLand":     h.workingDir,
			"config":        plugin.Config,
		},
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	// Get exported functions
	allocate := module.ExportedFunction("allocate")
	process := module.ExportedFunction("process")
	getOutputPtr := module.ExportedFunction("get_output_ptr")
	getOutputLen := module.ExportedFunction("get_output_len")

	if allocate == nil || process == nil {
		return nil, fmt.Errorf("WASM module missing required exports")
	}

	// Allocate memory for input
	results, err := allocate.Call(ctx, uint64(len(inputJSON)))
	if err != nil {
		return nil, fmt.Errorf("allocate memory: %w", err)
	}
	inputPtr := results[0]

	// Write input to memory
	mem := module.Memory()
	if mem == nil {
		return nil, fmt.Errorf("WASM module has no memory")
	}

	ok = mem.Write(uint32(inputPtr), inputJSON)
	if !ok {
		return nil, fmt.Errorf("write to WASM memory failed")
	}

	// Call process function
	_, err = process.Call(ctx, uint64(len(inputJSON)))
	if err != nil {
		return nil, fmt.Errorf("WASM process failed: %w", err)
	}

	// Get output
	if getOutputPtr != nil && getOutputLen != nil {
		ptrResults, err := getOutputPtr.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("get output pointer: %w", err)
		}
		lenResults, err := getOutputLen.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("get output length: %w", err)
		}

		outputPtr := uint32(ptrResults[0])
		outputLen := uint32(lenResults[0])

		outputBytes, ok := mem.Read(outputPtr, outputLen)
		if !ok {
			return nil, fmt.Errorf("read from WASM memory failed")
		}

		// Parse output
		var output struct {
			Success bool        `json:"success"`
			Result  interface{} `json:"result"`
			Error   string      `json:"error"`
		}
		if err := json.Unmarshal(outputBytes, &output); err != nil {
			return nil, fmt.Errorf("parse WASM output: %w", err)
		}

		if !output.Success {
			return nil, fmt.Errorf("WASM error: %s", output.Error)
		}

		return output.Result, nil
	}

	return nil, fmt.Errorf("WASM module missing output functions")
}

// Close closes the WASM runtime
func (h *WASMPluginHandler) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all modules
	for _, module := range h.modules {
		module.Close(ctx)
	}
	h.modules = make(map[string]api.Module)

	// Close runtime
	return h.runtime.Close(ctx)
}

// Helper functions for cross-platform compatibility
func isAbs(path string) bool {
	return filepath.IsAbs(path)
}

func joinPath(elem ...string) string {
	return filepath.Join(elem...)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
