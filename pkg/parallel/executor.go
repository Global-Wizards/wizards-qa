package parallel

import (
	"context"
	"fmt"
	"sync"
)

// Task represents a unit of work
type Task func() error

// Result represents the result of a task execution
type Result struct {
	Index int
	Error error
}

// Execute runs tasks in parallel with a max concurrency limit
func Execute(ctx context.Context, tasks []Task, maxConcurrency int) []error {
	if maxConcurrency <= 0 {
		maxConcurrency = len(tasks)
	}
	
	results := make([]error, len(tasks))
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	
	for i, task := range tasks {
		wg.Add(1)
		
		go func(index int, t Task) {
			defer wg.Done()
			
			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				results[index] = ctx.Err()
				return
			}
			
			// Execute task
			results[index] = t()
			
			// Release semaphore
			<-semaphore
		}(i, task)
	}
	
	wg.Wait()
	return results
}

// Map applies a function to each item in parallel
func Map[T any, R any](ctx context.Context, items []T, fn func(T) (R, error), maxConcurrency int) ([]R, []error) {
	if maxConcurrency <= 0 {
		maxConcurrency = len(items)
	}
	
	results := make([]R, len(items))
	errors := make([]error, len(items))
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	
	for i, item := range items {
		wg.Add(1)
		
		go func(index int, it T) {
			defer wg.Done()
			
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				errors[index] = ctx.Err()
				return
			}
			
			results[index], errors[index] = fn(it)
			<-semaphore
		}(i, item)
	}
	
	wg.Wait()
	return results, errors
}

// Batch processes items in parallel batches
type BatchProcessor[T any] struct {
	BatchSize    int
	MaxConcurrency int
	ProcessBatch func([]T) error
}

// Process executes batch processing
func (bp *BatchProcessor[T]) Process(ctx context.Context, items []T) error {
	if bp.BatchSize <= 0 {
		bp.BatchSize = 10
	}
	
	// Create batches
	var batches [][]T
	for i := 0; i < len(items); i += bp.BatchSize {
		end := i + bp.BatchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}
	
	// Process batches in parallel
	tasks := make([]Task, len(batches))
	for i, batch := range batches {
		b := batch // Capture for closure
		tasks[i] = func() error {
			return bp.ProcessBatch(b)
		}
	}
	
	errors := Execute(ctx, tasks, bp.MaxConcurrency)
	
	// Collect errors
	var errs []error
	for _, err := range errors {
		if err != nil {
			errs = append(errs, err)
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("batch processing failed with %d errors: %v", len(errs), errs[0])
	}
	
	return nil
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers int
	tasks   chan Task
	results chan error
	wg      sync.WaitGroup
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	return &WorkerPool{
		workers: workers,
		tasks:   make(chan Task),
		results: make(chan error),
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx)
	}
}

// Submit submits a task to the pool
func (wp *WorkerPool) Submit(task Task) {
	wp.tasks <- task
}

// Wait waits for all tasks to complete
func (wp *WorkerPool) Wait() {
	close(wp.tasks)
	wp.wg.Wait()
	close(wp.results)
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan error {
	return wp.results
}

func (wp *WorkerPool) worker(ctx context.Context) {
	defer wp.wg.Done()
	
	for {
		select {
		case task, ok := <-wp.tasks:
			if !ok {
				return
			}
			wp.results <- task()
		case <-ctx.Done():
			return
		}
	}
}
