// Package workerpool provides a simple bounded concurrency pool for
// running tasks in parallel with a fixed number of workers.
//
// Patterns covered:
//
//	p := workerpool.New(10)           // 10 concurrent workers
//	for _, item := range items {
//	    item := item
//	    p.Submit(func() { process(item) })
//	}
//	p.Wait()                          // block until all submitted tasks finish
//
// Zero third-party deps.
package workerpool

import (
	"context"
	"sync"
)

// Pool is a fixed-size goroutine pool for running func() tasks.
//
// After Close (or Wait) the pool cannot be reused.
type Pool struct {
	tasks chan func()
	wg    sync.WaitGroup

	closeMu sync.Mutex
	closed  bool
}

// New returns a Pool with `workers` goroutines accepting tasks from an
// unbuffered queue.
func New(workers int) *Pool {
	if workers < 1 {
		workers = 1
	}
	p := &Pool{tasks: make(chan func())}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		func() {
			defer func() {
				// swallow panics so one bad task doesn't kill a worker
				_ = recover()
			}()
			task()
		}()
	}
}

// Submit enqueues a task. Blocks if all workers are busy.
// Panics if called after Wait/Close.
func (p *Pool) Submit(task func()) {
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		panic("workerpool: Submit on closed pool")
	}
	p.closeMu.Unlock()
	p.tasks <- task
}

// SubmitCtx is like Submit but returns ctx.Err() if the context is
// cancelled before the task can be enqueued.
func (p *Pool) SubmitCtx(ctx context.Context, task func()) error {
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		return ErrClosed
	}
	p.closeMu.Unlock()
	select {
	case p.tasks <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Wait closes the task queue, waits for all workers to finish, then returns.
// The pool cannot be reused after Wait.
func (p *Pool) Wait() {
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		return
	}
	p.closed = true
	close(p.tasks)
	p.closeMu.Unlock()
	p.wg.Wait()
}

// ErrClosed is returned by SubmitCtx after the pool has been closed.
type errClosed struct{}

func (errClosed) Error() string { return "workerpool: closed" }

// ErrClosed sentinel.
var ErrClosed error = errClosed{}
