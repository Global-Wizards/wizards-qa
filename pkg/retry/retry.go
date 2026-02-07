package retry

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Config defines retry configuration
type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultConfig returns sensible retry defaults
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// IsRetryable checks if an error is retryable
type IsRetryable func(error) bool

// Do executes a function with exponential backoff retry
func Do(ctx context.Context, config *Config, fn func() error) error {
	return doRetry(ctx, config, nil, fn)
}

// DoWithRetryable executes with custom retry logic
func DoWithRetryable(ctx context.Context, config *Config, isRetryable IsRetryable, fn func() error) error {
	return doRetry(ctx, config, isRetryable, fn)
}

// DoWithBackoff executes with custom backoff calculation
func DoWithBackoff(ctx context.Context, maxAttempts int, backoff func(attempt int) time.Duration, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == maxAttempts {
			break
		}

		delay := backoff(attempt)
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// doRetry is the shared retry loop used by Do and DoWithRetryable.
func doRetry(ctx context.Context, config *Config, isRetryable IsRetryable, fn func() error) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable (when a checker is provided)
		if isRetryable != nil && !isRetryable(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		if attempt == config.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// ExponentialBackoff calculates exponential backoff delay
func ExponentialBackoff(attempt int, initial, max time.Duration) time.Duration {
	delay := time.Duration(float64(initial) * math.Pow(2, float64(attempt-1)))
	if delay > max {
		delay = max
	}
	return delay
}

// LinearBackoff calculates linear backoff delay
func LinearBackoff(attempt int, base time.Duration) time.Duration {
	return time.Duration(attempt) * base
}
