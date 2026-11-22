// Package pool processes tasks asynchronously with a fixed number of
// worker goroutines reading task IDs from a bounded channel.
package pool

import (
	"errors"
	"fmt"
	"sync"

	"textanalyzer-cp1/task"
)

var (
	ErrQueueFull     = errors.New("pool: queue is full")
	ErrPoolClosed    = errors.New("pool: shut down, not accepting tasks")
	ErrInvalidConfig = errors.New("pool: need at least 2 workers and queue capacity of at least 1")
)

type Pool struct {
	store *task.Store
	proc  task.Processor
	queue chan string // owned by Pool: only Submit sends, only Shutdown closes
	wg    sync.WaitGroup

	mu     sync.Mutex // guards closed and every send/close on queue
	closed bool
}

// New creates the pool and starts its workers immediately.
func New(store *task.Store, proc task.Processor, workers, queueCap int) (*Pool, error) {
	if workers < 2 || queueCap < 1 {
		return nil, fmt.Errorf("%w: workers=%d, queueCap=%d", ErrInvalidConfig, workers, queueCap)
	}
	p := &Pool{store: store, proc: proc, queue: make(chan string, queueCap)}
	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go p.worker()
	}
	return p, nil
}

// worker keeps taking task IDs until the queue is closed and drained.
func (p *Pool) worker() {
	defer p.wg.Done()
	for id := range p.queue {
		// Errors are already recorded on the task (status Failed); a
		// Queued->Processing failure cannot happen because only Submit
		// puts IDs in the queue, right after marking them Queued.
		_, _ = p.store.Process(id, task.StatusQueued, p.proc)
	}
}

// Submit marks a Pending task as Queued and puts it in the queue without
// waiting for it to be processed. It never blocks: a full queue returns
// ErrQueueFull and the task goes back to Pending.
func (p *Pool) Submit(id string) error {
	// Holding mu across the closed check and the send means Shutdown cannot
	// close the channel in between, so "send on closed channel" is impossible.
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrPoolClosed
	}
	if _, err := p.store.Transition(id, task.StatusPending, task.StatusQueued); err != nil {
		return err
	}
	select {
	case p.queue <- id:
		return nil
	default:
		// The ID never entered the queue, so no worker can touch it; undo.
		_, _ = p.store.Transition(id, task.StatusQueued, task.StatusPending)
		return ErrQueueFull
	}
}

// Shutdown stops accepting tasks, lets workers finish everything already
// queued, and waits for them to exit. Safe to call more than once.
func (p *Pool) Shutdown() {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.queue)
	}
	p.mu.Unlock()
	p.wg.Wait()
}
