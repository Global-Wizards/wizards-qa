package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoSuccess(t *testing.T) {
	cfg := &Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond, Multiplier: 2}
	calls := 0

	err := Do(context.Background(), cfg, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDoRetryThenSucceed(t *testing.T) {
	cfg := &Config{MaxAttempts: 5, InitialDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond, Multiplier: 2}
	calls := 0

	err := Do(context.Background(), cfg, func() error {
		calls++
		if calls < 3 {
			return errors.New("fail")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoExhausted(t *testing.T) {
	cfg := &Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond, Multiplier: 2}
	calls := 0

	err := Do(context.Background(), cfg, func() error {
		calls++
		return errors.New("always fails")
	})

	if err == nil {
		t.Fatal("expected error after exhausting attempts")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoWithRetryable_NonRetryable(t *testing.T) {
	cfg := &Config{MaxAttempts: 5, InitialDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond, Multiplier: 2}
	calls := 0
	permanent := errors.New("permanent error")

	err := DoWithRetryable(context.Background(), cfg, func(e error) bool {
		return e != permanent
	}, func() error {
		calls++
		return permanent
	})

	if err == nil {
		t.Fatal("expected error for non-retryable")
	}
	if calls != 1 {
		t.Errorf("expected 1 call for non-retryable, got %d", calls)
	}
}

func TestDoContextCancelled(t *testing.T) {
	cfg := &Config{MaxAttempts: 100, InitialDelay: 50 * time.Millisecond, MaxDelay: time.Second, Multiplier: 2}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, cfg, func() error {
		calls++
		return errors.New("fail")
	})

	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestDoNilConfig(t *testing.T) {
	calls := 0
	err := Do(context.Background(), nil, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestExponentialBackoff(t *testing.T) {
	d1 := ExponentialBackoff(1, 100*time.Millisecond, 10*time.Second)
	d2 := ExponentialBackoff(2, 100*time.Millisecond, 10*time.Second)
	d3 := ExponentialBackoff(3, 100*time.Millisecond, 10*time.Second)

	if d1 != 100*time.Millisecond {
		t.Errorf("attempt 1: got %v, want 100ms", d1)
	}
	if d2 != 200*time.Millisecond {
		t.Errorf("attempt 2: got %v, want 200ms", d2)
	}
	if d3 != 400*time.Millisecond {
		t.Errorf("attempt 3: got %v, want 400ms", d3)
	}
}

func TestExponentialBackoffCapped(t *testing.T) {
	d := ExponentialBackoff(10, 100*time.Millisecond, 5*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected capped at 5s, got %v", d)
	}
}

func TestLinearBackoff(t *testing.T) {
	d1 := LinearBackoff(1, 100*time.Millisecond)
	d3 := LinearBackoff(3, 100*time.Millisecond)

	if d1 != 100*time.Millisecond {
		t.Errorf("attempt 1: got %v, want 100ms", d1)
	}
	if d3 != 300*time.Millisecond {
		t.Errorf("attempt 3: got %v, want 300ms", d3)
	}
}

func TestDoWithBackoff(t *testing.T) {
	calls := 0
	err := DoWithBackoff(context.Background(), 3, func(attempt int) time.Duration {
		return time.Millisecond
	}, func() error {
		calls++
		if calls < 2 {
			return errors.New("fail")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != time.Second {
		t.Errorf("InitialDelay = %v, want 1s", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", cfg.MaxDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", cfg.Multiplier)
	}
}
