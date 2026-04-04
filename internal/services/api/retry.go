// Package api provides HTTP client for Claude API communication
package api

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []string
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"overloaded_error",
			"rate_limit_error",
			"api_error",
		},
	}
}

// RetryPolicy implements retry logic
type RetryPolicy struct {
	config *RetryConfig
	rand   *rand.Rand
}

// NewRetryPolicy creates a new retry policy
func NewRetryPolicy(config *RetryConfig) *RetryPolicy {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryPolicy{
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Execute executes a function with retry logic
func (p *RetryPolicy) Execute(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !p.isRetryable(err) {
			return err
		}

		// Check if we have more retries
		if attempt >= p.config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := p.calculateDelay(attempt)

		// Wait or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// isRetryable checks if an error is retryable
func (p *RetryPolicy) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check API error
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsRetryable()
	}

	// Check error type string
	errStr := err.Error()
	for _, retryable := range p.config.RetryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// calculateDelay calculates the delay with exponential backoff and jitter
func (p *RetryPolicy) calculateDelay(attempt int) time.Duration {
	// Exponential backoff
	delay := float64(p.config.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= p.config.BackoffFactor
	}

	// Cap at max delay
	if delay > float64(p.config.MaxDelay) {
		delay = float64(p.config.MaxDelay)
	}

	// Add jitter (0.5 to 1.5)
	jitter := 0.5 + p.rand.Float64()
	delay *= jitter

	return time.Duration(delay)
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryableError wraps an error to indicate it's retryable
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a retryable error
func NewRetryableError(err error) error {
	return &RetryableError{Err: err}
}

// IsRetryable checks if an error is a retryable error
func IsRetryable(err error) bool {
	_, ok := err.(*RetryableError)
	return ok
}

// TransientError represents a transient error that should be retried
type TransientError struct {
	Message string
	Cause   error
}

func (e *TransientError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *TransientError) Unwrap() error {
	return e.Cause
}

// NewTransientError creates a transient error
func NewTransientError(message string, cause error) error {
	return &TransientError{
		Message: message,
		Cause:   cause,
	}
}
