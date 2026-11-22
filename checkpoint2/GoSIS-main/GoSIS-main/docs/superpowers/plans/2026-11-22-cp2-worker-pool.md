# CP2 Worker Pool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a bounded-queue worker pool to the CP1 Text Analyzer in `SIS2/`.

**Architecture:** `task.Store` becomes the single synchronized owner of task state (RWMutex, returns snapshot copies,
atomic transitions, one `Process` path used by both sync service and workers). New `pool` package: buffered `chan string`
of IDs, N≥2 workers, non-blocking Submit, mutex-guarded close + WaitGroup shutdown.

**Tech Stack:** Go 1.27 stdlib only (`sync`, `errors`, `testing`).

**Spec:** `docs/superpowers/specs/2026-11-22-cp2-worker-pool-design.md` (requirements: `message.txt`)

## Global Constraints
- Module stays `textanalyzer-cp1`; all CP1 tests keep passing (only pointer-identity assert may change to value compare).
- `gofmt -l .` empty; `go vet ./...`, `go test ./...`, `go test -race ./...` pass.
- No panics for expected errors; no `time.Sleep` to make tests pass (timeouts only as hang guards).
- Workers ≥ 2; queue finite; full queue ⇒ `ErrQueueFull`, task stays Pending.

## Review Focus
- Submit concurrent with Shutdown → no panic; every nil-returning submit ends Completed. (Task 2: `TestPool_SubmitDuringShutdown`)
- Re-submit of Queued/Processing/Completed task → `ErrInvalidTransition`. (Task 2: `TestPool_InvalidSubmissions`)
- Sync `ProcessTask` on a Queued task must not double-process → `ErrInvalidTransition`. (Task 1: `TestStore_Process_WrongState`)
- Caller mutating a returned snapshot must not change stored task. (Task 1: `TestStore_SnapshotsAreCopies`)
- Shutdown called twice / concurrently → no panic, returns. (Task 2: `TestPool_ShutdownDrainsAndRejects`)

---

### Task 1: Synchronized store + Queued state

**Files:** Modify `SIS2/task/task.go`, `SIS2/task/store.go`, `SIS2/service/service.go`, `SIS2/task/store_test.go`, `SIS2/go.mod`

**Produces:**
- `StatusQueued Status = "queued"`
- `(*Store).Create(input string) (*Task, error)`, `Get(id) (*Task, error)`, `All() []*Task` — return **copies**
- `(*Store).Transition(id string, from, to Status) (*Task, error)` — atomic check-and-set; `ErrTaskNotFound` / `ErrInvalidTransition`
- `(*Store).Process(id string, from Status, p Processor) (*Task, error)` — from→Processing (lock), `p.Process` (no lock), Completed/Failed (lock); on processor error returns snapshot + `fmt.Errorf("process task %s: %w", id, err)`
- `service.ProcessTask(id)` = `store.Process(id, StatusPending, processor)`

- [ ] Tests: `TestStore_Transition` (Pending→Queued ok; Queued→Queued err; missing id ErrTaskNotFound), `TestStore_Process_WrongState` (Queued task with from=Pending → ErrInvalidTransition, status unchanged), `TestStore_SnapshotsAreCopies`, `TestStore_ConcurrentCreateGet` (100 goroutines, unique IDs, Len==100); CP1 `got != created` → `*got != *created`.
- [ ] Run `go test -race ./...` → FAIL (undefined)
- [ ] Implement; `go mod edit -go=1.27`
- [ ] Run `go test -race ./...` → PASS; commit `feat(task): synchronized store, queued state, shared Process path`

### Task 2: Worker pool

**Files:** Create `SIS2/pool/pool.go`, `SIS2/pool/pool_test.go`

**Consumes:** Task 1 `Transition`, `Process`, `Get`.
**Produces:** `ErrQueueFull`, `ErrPoolClosed`, `ErrInvalidConfig`; `New(store *task.Store, p task.Processor, workers, queueCap int) (*Pool, error)` (starts workers; workers<2 or cap<1 → ErrInvalidConfig); `(*Pool).Submit(id string) error`; `(*Pool).Shutdown()`.

Submit under `p.mu`: closed → ErrPoolClosed; Transition Pending→Queued; `select` non-blocking send; default → Transition back Queued→Pending, ErrQueueFull.
Worker: `for id := range queue { store.Process(id, StatusQueued, proc) }`. Shutdown: under mu close once; `wg.Wait()`.

- [ ] Tests (gate processor: `started chan string` (buffered), `release chan struct{}`):
  `TestNew_InvalidConfig`, `TestPool_ProcessesConcurrently` (2 workers both report started before release; both Get==Processing),
  `TestPool_CorrectResults` (real analyzer, inputs with 1..5 words → each Completed with WordCount==n),
  `TestPool_QueueFull` (2 workers, cap 1: A,B in flight, C queued, D → ErrQueueFull & Pending; after release A,B,C Completed),
  `TestPool_InvalidSubmissions`, `TestPool_ShutdownDrainsAndRejects`, `TestPool_SubmitDuringShutdown`, `TestPool_ConcurrentSubmitAndGet`.
- [ ] Run → FAIL; implement; run `go test -race ./...` → PASS; commit `feat(pool): bounded worker pool`

### Task 3: Demo + README

**Files:** Modify `SIS2/main.go`, `SIS2/README.md`
- [ ] main: 3 workers, cap 10; create 6 tasks, submit, show duplicate-submit error, Shutdown, print results, show submit-after-shutdown error; replace `must`/panic with `log.Fatal`.
- [ ] README: purpose, submit flow, workers=3, cap=10, full-queue behavior, lifecycle table with Queued, synchronization (what each lock guards, lock order pool.mu→store.mu), shutdown, channel ownership, run commands; drop Go-version note.
- [ ] Run full checklist (`gofmt -l .`, vet, test, test -race, `go run .`); commit `docs: CP2 README and demo`
