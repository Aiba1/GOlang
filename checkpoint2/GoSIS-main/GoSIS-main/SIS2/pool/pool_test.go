package pool

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"textanalyzer-cp1/analyzer"
	"textanalyzer-cp1/task"
)

// gateProc reports every task it starts on started and then blocks until
// release is closed. This lets tests hold workers inside Process and
// observe the pool at a known point without time.Sleep.
type gateProc struct {
	started chan string
	release chan struct{}
}

func newGate() *gateProc {
	return &gateProc{started: make(chan string, 100), release: make(chan struct{})}
}

func (g *gateProc) Process(in string) (task.Result, error) {
	g.started <- in
	<-g.release
	return analyzer.New().Process(in)
}

// waitStarted waits until n tasks are inside Process. The timeout is only
// a guard so a broken pool fails the test instead of hanging it.
func waitStarted(t *testing.T, g *gateProc, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-g.started:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of %d tasks started processing", i, n)
		}
	}
}

func mustCreate(t *testing.T, s *task.Store, input string) string {
	t.Helper()
	created, err := s.Create(input)
	if err != nil {
		t.Fatalf("Create(%q): %v", input, err)
	}
	return created.ID
}

func status(t *testing.T, s *task.Store, id string) task.Status {
	t.Helper()
	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get(%q): %v", id, err)
	}
	return got.Status
}

func TestNew_InvalidConfig(t *testing.T) {
	tests := []struct{ workers, capacity int }{{1, 5}, {0, 5}, {2, 0}, {-1, -1}}
	for _, tt := range tests {
		if _, err := New(task.NewStore(), analyzer.New(), tt.workers, tt.capacity); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("New(workers=%d, cap=%d) error = %v, want ErrInvalidConfig", tt.workers, tt.capacity, err)
		}
	}
}

func TestNew_Startup(t *testing.T) {
	s := task.NewStore()
	p, err := New(s, analyzer.New(), 2, 1)
	if err != nil || p == nil {
		t.Fatalf("New(2 workers, cap 1) = %v, %v; want a running pool", p, err)
	}
	id := mustCreate(t, s, "started")
	if err := p.Submit(id); err != nil {
		t.Fatalf("Submit on fresh pool: %v", err)
	}
	p.Shutdown()
	if st := status(t, s, id); st != task.StatusCompleted {
		t.Errorf("status = %v, want completed (workers did not run)", st)
	}
}

func TestPool_ProcessesConcurrently(t *testing.T) {
	s := task.NewStore()
	g := newGate()
	p, err := New(s, g, 2, 5)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	a, b := mustCreate(t, s, "alpha"), mustCreate(t, s, "beta")
	for _, id := range []string{a, b} {
		if err := p.Submit(id); err != nil {
			t.Fatalf("Submit(%s): %v", id, err)
		}
	}

	// Neither task can finish before release is closed, so if both have
	// started, two workers are inside Process at the same time.
	waitStarted(t, g, 2)
	for _, id := range []string{a, b} {
		if st := status(t, s, id); st != task.StatusProcessing {
			t.Errorf("%s status = %v while held, want processing", id, st)
		}
	}

	close(g.release)
	p.Shutdown()
	for _, id := range []string{a, b} {
		if st := status(t, s, id); st != task.StatusCompleted {
			t.Errorf("%s status = %v after shutdown, want completed", id, st)
		}
	}
}

func TestPool_CorrectResults(t *testing.T) {
	s := task.NewStore()
	p, err := New(s, analyzer.New(), 3, 10)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	want := map[string]int{} // task ID -> expected word count
	for n := 1; n <= 8; n++ {
		id := mustCreate(t, s, strings.Repeat("go ", n))
		want[id] = n
		if err := p.Submit(id); err != nil {
			t.Fatalf("Submit(%s): %v", id, err)
		}
	}
	p.Shutdown()

	for id, words := range want {
		got, _ := s.Get(id)
		if got.Status != task.StatusCompleted {
			t.Errorf("%s status = %v, want completed", id, got.Status)
			continue
		}
		if r := got.Result.(analyzer.Result); r.WordCount != words {
			t.Errorf("%s WordCount = %d, want %d (result attached to wrong task?)", id, r.WordCount, words)
		}
	}
}

func TestPool_QueueFull(t *testing.T) {
	s := task.NewStore()
	g := newGate()
	p, err := New(s, g, 2, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	a, b, c, d := mustCreate(t, s, "a"), mustCreate(t, s, "b"), mustCreate(t, s, "c"), mustCreate(t, s, "d")

	// Capacity is 1, so submit one at a time and wait until a worker has
	// taken it out of the queue before the next one.
	for _, id := range []string{a, b} {
		if err := p.Submit(id); err != nil {
			t.Fatalf("Submit(%s): %v", id, err)
		}
		waitStarted(t, g, 1)
	}
	// Both workers are now busy and the queue is empty.
	if err := p.Submit(c); err != nil {
		t.Fatalf("Submit(c) into empty queue: %v", err)
	}
	if err := p.Submit(d); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("Submit(d) error = %v, want ErrQueueFull", err)
	}
	if st := status(t, s, d); st != task.StatusPending {
		t.Errorf("rejected task status = %v, want pending", st)
	}
	if st := status(t, s, c); st != task.StatusQueued {
		t.Errorf("queued task status = %v, want queued", st)
	}

	close(g.release)
	p.Shutdown()
	for _, id := range []string{a, b, c} {
		if st := status(t, s, id); st != task.StatusCompleted {
			t.Errorf("%s status = %v, want completed", id, st)
		}
	}
	if st := status(t, s, d); st != task.StatusPending {
		t.Errorf("rejected task status after shutdown = %v, want pending", st)
	}
}

func TestPool_InvalidSubmissions(t *testing.T) {
	s := task.NewStore()
	g := newGate()
	p, err := New(s, g, 2, 5)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	a, b, c := mustCreate(t, s, "a"), mustCreate(t, s, "b"), mustCreate(t, s, "c")

	if err := p.Submit("task-404"); !errors.Is(err, task.ErrTaskNotFound) {
		t.Errorf("Submit(missing) error = %v, want ErrTaskNotFound", err)
	}

	_ = p.Submit(a)
	_ = p.Submit(b)
	waitStarted(t, g, 2)
	if err := p.Submit(a); !errors.Is(err, task.ErrInvalidTransition) {
		t.Errorf("Submit(processing) error = %v, want ErrInvalidTransition", err)
	}

	_ = p.Submit(c) // waits in queue: both workers are held
	if err := p.Submit(c); !errors.Is(err, task.ErrInvalidTransition) {
		t.Errorf("Submit(queued) error = %v, want ErrInvalidTransition", err)
	}

	close(g.release)
	p.Shutdown()

	p2, _ := New(s, analyzer.New(), 2, 5)
	defer p2.Shutdown()
	if err := p2.Submit(a); !errors.Is(err, task.ErrInvalidTransition) {
		t.Errorf("Submit(completed) error = %v, want ErrInvalidTransition", err)
	}
}

func TestPool_ShutdownDrainsAndRejects(t *testing.T) {
	s := task.NewStore()
	p, err := New(s, analyzer.New(), 2, 10)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var ids []string
	for i := 0; i < 10; i++ {
		id := mustCreate(t, s, fmt.Sprintf("text number %d", i))
		ids = append(ids, id)
		if err := p.Submit(id); err != nil {
			t.Fatalf("Submit(%s): %v", id, err)
		}
	}

	done := make(chan struct{})
	go func() {
		p.Shutdown()
		p.Shutdown() // second call must be a harmless no-op
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not return: workers did not terminate")
	}

	for _, id := range ids {
		if st := status(t, s, id); st != task.StatusCompleted {
			t.Errorf("%s status = %v, want completed (accepted task lost)", id, st)
		}
	}

	late := mustCreate(t, s, "too late")
	if err := p.Submit(late); !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Submit after shutdown error = %v, want ErrPoolClosed", err)
	}
	if st := status(t, s, late); st != task.StatusPending {
		t.Errorf("rejected task status = %v, want pending", st)
	}
}

func TestPool_SubmitDuringShutdown(t *testing.T) {
	s := task.NewStore()
	p, err := New(s, analyzer.New(), 2, 50)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	const n = 50
	ids := make([]string, n)
	for i := range ids {
		ids[i] = mustCreate(t, s, "some words here")
	}
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = p.Submit(ids[i])
		}()
	}
	p.Shutdown() // races with the submitters; must not panic
	wg.Wait()

	for i, id := range ids {
		st := status(t, s, id)
		switch {
		case errs[i] == nil && st != task.StatusCompleted:
			t.Errorf("%s accepted but status = %v, want completed", id, st)
		case errors.Is(errs[i], ErrPoolClosed) && st != task.StatusPending:
			t.Errorf("%s rejected but status = %v, want pending", id, st)
		case errs[i] != nil && !errors.Is(errs[i], ErrPoolClosed):
			t.Errorf("%s unexpected error %v", id, errs[i])
		}
	}
}

func TestPool_ConcurrentSubmitAndGet(t *testing.T) {
	s := task.NewStore()
	p, err := New(s, analyzer.New(), 4, 100)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	const n = 100
	ids := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := s.Create("one two three") // not mustCreate: t.Fatal is not allowed off the test goroutine
			if err != nil {
				t.Error(err)
				return
			}
			id := created.ID
			if err := p.Submit(id); err != nil {
				t.Errorf("Submit(%s): %v", id, err)
				return
			}
			// Read while workers may be writing the same task.
			if _, err := s.Get(id); err != nil {
				t.Errorf("Get(%s): %v", id, err)
			}
			_ = s.All()
			ids <- id
		}()
	}
	wg.Wait()
	p.Shutdown()
	close(ids)

	count := 0
	for id := range ids {
		count++
		got, _ := s.Get(id)
		if got.Status != task.StatusCompleted || got.Result.(analyzer.Result).WordCount != 3 {
			t.Errorf("%s = %v/%v, want completed with 3 words", id, got.Status, got.Result)
		}
	}
	if count != n {
		t.Errorf("processed %d tasks, want %d", count, n)
	}
}
