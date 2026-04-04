package hooks

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// AsyncRegistry manages asynchronous hook executions
type AsyncRegistry struct {
	mu      sync.RWMutex
	pending map[string]*pendingAsyncHook
}

type pendingAsyncHook struct {
	info      AsyncHookInfo
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	output    *HookOutput
	err       error
}

// NewAsyncRegistry creates a new async hook registry
func NewAsyncRegistry() *AsyncRegistry {
	return &AsyncRegistry{
		pending: make(map[string]*pendingAsyncHook),
	}
}

// Register registers a new async hook and returns its process ID
func (r *AsyncRegistry) Register(info AsyncHookInfo) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	processID := generateProcessID()

	ctx, cancel := context.WithTimeout(context.Background(), info.Timeout)

	r.pending[processID] = &pendingAsyncHook{
		info:   info,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	return processID
}

// SetResult sets the result for a pending async hook
func (r *AsyncRegistry) SetResult(processID string, output *HookOutput, err error) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pending[processID]
	if !ok {
		return false
	}

	p.output = output
	p.err = err
	close(p.done)

	return true
}

// Wait waits for a specific async hook to complete
func (r *AsyncRegistry) Wait(processID string, timeout time.Duration) (*HookOutput, error) {
	r.mu.RLock()
	p, ok := r.pending[processID]
	r.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	select {
	case <-p.done:
		return p.output, p.err
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// CheckCompleted returns all completed async hooks
func (r *AsyncRegistry) CheckCompleted() []*AsyncHookResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*AsyncHookResult

	for processID, p := range r.pending {
		select {
		case <-p.done:
			results = append(results, &AsyncHookResult{
				ProcessID: processID,
				HookID:    p.info.HookID,
				Output:    p.output,
				Error:     p.err,
			})
		default:
			// Still running
		}
	}

	return results
}

// Cancel cancels a specific async hook
func (r *AsyncRegistry) Cancel(processID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pending[processID]
	if !ok {
		return false
	}

	p.cancel()
	return true
}

// CancelAll cancels all pending async hooks
func (r *AsyncRegistry) CancelAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range r.pending {
		p.cancel()
	}
}

// Remove removes a completed async hook from the registry
func (r *AsyncRegistry) Remove(processID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.pending, processID)
}

// Finalize cleans up all pending async hooks
func (r *AsyncRegistry) Finalize() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range r.pending {
		p.cancel()
	}

	// Clear all pending
	r.pending = make(map[string]*pendingAsyncHook)
}

// GetPending returns information about all pending async hooks
func (r *AsyncRegistry) GetPending() []AsyncHookInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var infos []AsyncHookInfo
	for _, p := range r.pending {
		infos = append(infos, p.info)
	}

	return infos
}

// Count returns the number of pending async hooks
func (r *AsyncRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.pending)
}

// generateProcessID generates a unique process ID
func generateProcessID() string {
	return fmt.Sprintf("async_%d_%d", time.Now().UnixNano(), os.Getpid())
}
