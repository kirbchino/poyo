// Package executor provides tool execution with concurrency support
package executor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kirbchino/poyo/internal/tools"
)

// ConcurrentSafeTools is a set of tools that are safe to execute concurrently
var ConcurrentSafeTools = map[string]bool{
	"Read":    true,
	"Glob":    true,
	"Grep":    true,
	"WebFetch": true,
	"WebSearch": true,
	"TodoWrite": true,
}

// ToolExecutor executes tools with concurrency management
type ToolExecutor struct {
	registry      *tools.Registry
	permChecker   PermissionChecker
	maxConcurrent int
	semaphore     chan struct{}
	mu            sync.RWMutex
	activeCount   int32
}

// PermissionChecker checks permissions for tool execution
type PermissionChecker interface {
	Check(ctx context.Context, toolName string, input map[string]interface{}) (bool, string, error)
}

// ExecutionResult represents the result of a tool execution
type ExecutionResult struct {
	ToolName    string
	Success     bool
	Output      interface{}
	Error       error
	Duration    time.Duration
	Blocked     bool
	BlockReason string
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(registry *tools.Registry, maxConcurrent int) *ToolExecutor {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	return &ToolExecutor{
		registry:      registry,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// SetPermissionChecker sets the permission checker
func (e *ToolExecutor) SetPermissionChecker(checker PermissionChecker) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.permChecker = checker
}

// Execute executes a single tool
func (e *ToolExecutor) Execute(ctx context.Context, toolName string, input map[string]interface{}, toolCtx *tools.ToolUseContext) *ExecutionResult {
	start := time.Now()
	result := &ExecutionResult{
		ToolName: toolName,
	}

	// Get tool
	tool := e.registry.Get(toolName)
	if tool == nil {
		result.Error = fmt.Errorf("unknown tool: %s", toolName)
		return result
	}

	// Check permission
	if e.permChecker != nil {
		allowed, reason, err := e.permChecker.Check(ctx, toolName, input)
		if err != nil {
			result.Error = fmt.Errorf("permission check failed: %w", err)
			return result
		}
		if !allowed {
			result.Blocked = true
			result.BlockReason = reason
			return result
		}
	}

	// Acquire semaphore for concurrency control
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		result.Error = ctx.Err()
		return result
	}

	atomic.AddInt32(&e.activeCount, 1)
	defer atomic.AddInt32(&e.activeCount, -1)

	// Execute tool
	output, err := tool.Call(ctx, input, toolCtx, nil, nil)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err
		return result
	}

	result.Success = true
	result.Output = output
	return result
}

// ExecuteBatch executes multiple tools with concurrency management
func (e *ToolExecutor) ExecuteBatch(ctx context.Context, calls []ToolCall, toolCtx *tools.ToolUseContext) []*ExecutionResult {
	results := make([]*ExecutionResult, len(calls))

	// Separate concurrent-safe and unsafe tools
	var safeCalls, unsafeCalls []int
	for i, call := range calls {
		if ConcurrentSafeTools[call.Name] {
			safeCalls = append(safeCalls, i)
		} else {
			unsafeCalls = append(unsafeCalls, i)
		}
	}

	var wg sync.WaitGroup

	// Execute concurrent-safe tools in parallel
	for _, idx := range safeCalls {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = e.Execute(ctx, calls[i].Name, calls[i].Input, toolCtx)
		}(idx)
	}

	// Wait for safe tools to complete
	wg.Wait()

	// Execute non-safe tools sequentially
	for _, idx := range unsafeCalls {
		results[idx] = e.Execute(ctx, calls[idx].Name, calls[idx].Input, toolCtx)
	}

	return results
}

// ExecuteParallel executes tools in parallel when safe
func (e *ToolExecutor) ExecuteParallel(ctx context.Context, calls []ToolCall, toolCtx *tools.ToolUseContext, progress chan<- ParallelProgress) []*ExecutionResult {
	results := make([]*ExecutionResult, len(calls))

	var completed int32
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c ToolCall) {
			defer wg.Done()

			result := e.Execute(ctx, c.Name, c.Input, toolCtx)
			results[idx] = result

			// Report progress
			if progress != nil {
				done := atomic.AddInt32(&completed, 1)
				progress <- ParallelProgress{
					Index:     idx,
					ToolName:  c.Name,
					Completed: int(done),
					Total:     len(calls),
					Success:   result.Success,
					Error:     result.Error,
				}
			}
		}(i, call)
	}

	wg.Wait()
	close(progress)
	return results
}

// ToolCall represents a tool call
type ToolCall struct {
	Name      string                 `json:"name"`
	Input     map[string]interface{} `json:"input"`
	ID        string                 `json:"id,omitempty"`
	DependsOn []string               `json:"depends_on,omitempty"`
}

// ParallelProgress represents progress of parallel execution
type ParallelProgress struct {
	Index     int
	ToolName  string
	Completed int
	Total     int
	Success   bool
	Error     error
}

// GetActiveCount returns the number of currently active executions
func (e *ToolExecutor) GetActiveCount() int {
	return int(atomic.LoadInt32(&e.activeCount))
}

// DependencyExecutor executes tools with dependency management
type DependencyExecutor struct {
	*ToolExecutor
}

// NewDependencyExecutor creates a new dependency-aware executor
func NewDependencyExecutor(registry *tools.Registry, maxConcurrent int) *DependencyExecutor {
	return &DependencyExecutor{
		ToolExecutor: NewToolExecutor(registry, maxConcurrent),
	}
}

// ExecuteWithDependencies executes tools respecting dependencies
func (e *DependencyExecutor) ExecuteWithDependencies(ctx context.Context, calls []ToolCall, toolCtx *tools.ToolUseContext) []*ExecutionResult {
	results := make([]*ExecutionResult, len(calls))

	// Execute in topological order
	executed := make(map[string]bool)
	for len(executed) < len(calls) {
		// Find tools whose dependencies are satisfied
		var ready []int
		for i, call := range calls {
			if executed[call.ID] {
				continue
			}

			// Check if all dependencies are executed
			allDepsSatisfied := true
			for _, depID := range call.DependsOn {
				if !executed[depID] {
					allDepsSatisfied = false
					break
				}
			}

			if allDepsSatisfied {
				ready = append(ready, i)
			}
		}

		if len(ready) == 0 {
			// Circular dependency or other issue
			break
		}

		// Execute ready tools
		var readyCalls []ToolCall
		for _, idx := range ready {
			readyCalls = append(readyCalls, calls[idx])
		}

		readyResults := e.ExecuteBatch(ctx, readyCalls, toolCtx)

		for i, idx := range ready {
			results[idx] = readyResults[i]
			executed[calls[idx].ID] = true
		}
	}

	// Mark remaining as failed (circular dependency)
	for i, result := range results {
		if result == nil {
			results[i] = &ExecutionResult{
				ToolName: calls[i].Name,
				Error:    fmt.Errorf("circular dependency or unsatisfied dependencies"),
			}
		}
	}

	return results
}

// RateLimitedExecutor wraps an executor with rate limiting
type RateLimitedExecutor struct {
	*ToolExecutor
	rateLimiter *RateLimiter
}

// RateLimiter provides rate limiting
type RateLimiter struct {
	tokens     chan struct{}
	refillRate time.Duration
	stop       chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:     make(chan struct{}, maxTokens),
		refillRate: refillRate,
		stop:       make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < maxTokens; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refill goroutine
	go rl.refill()

	return rl
}

// refill periodically refills tokens
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.refillRate)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Token bucket is full
			}
		}
	}
}

// Wait waits for a token
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// NewRateLimitedExecutor creates a new rate-limited executor
func NewRateLimitedExecutor(registry *tools.Registry, maxConcurrent int, rateLimit int, refillRate time.Duration) *RateLimitedExecutor {
	return &RateLimitedExecutor{
		ToolExecutor: NewToolExecutor(registry, maxConcurrent),
		rateLimiter:  NewRateLimiter(rateLimit, refillRate),
	}
}

// Execute executes with rate limiting
func (e *RateLimitedExecutor) Execute(ctx context.Context, toolName string, input map[string]interface{}, toolCtx *tools.ToolUseContext) *ExecutionResult {
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return &ExecutionResult{
			ToolName: toolName,
			Error:    err,
		}
	}
	return e.ToolExecutor.Execute(ctx, toolName, input, toolCtx)
}

// Stop stops the rate limiter
func (e *RateLimitedExecutor) Stop() {
	e.rateLimiter.Stop()
}
