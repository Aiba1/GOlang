# Text Analyzer — SIS Checkpoint 2 (concurrent)

Continuation of the Checkpoint 1 project for **Golang Application Development**.
A task is a request to analyze a piece of text. Checkpoint 2 adds concurrent
processing: tasks go into a bounded queue and a fixed worker pool processes them.

## Domain

A task's input is raw text. Processing (`analyzer.TextAnalyzer`, the same code as in CP1) computes:
word count, character count, character count without whitespace, the longest word and the average word length.

```
Input:  "Hello Go world"
Result: words=3 chars=14 chars_no_spaces=12 longest_word="Hello" avg_word_length=4.00
```

## Package layout

```
task/      Task, Status, Processor/Result interfaces, synchronized in-memory Store
analyzer/  TextAnalyzer — the domain processing (unchanged from CP1)
service/   TaskService — CreateTask / GetTask / ProcessTask (CP1 synchronous path)
pool/      Pool — bounded queue + worker goroutines (CP2 asynchronous path)
main.go    demo (go run .)
```

## Concurrency architecture

```
 CreateTask ──> Store (Pending)
                  │
 Submit(id) ──────┤ Pending→Queued (under store lock)
                  ▼
        queue: chan string, capacity 10   (full → ErrQueueFull, task back to Pending)
                  │
      ┌───────────┼───────────┐
      ▼           ▼           ▼
  worker 1    worker 2    worker 3     for id := range queue
      │           │           │
      └─── Store.Process: Queued→Processing (lock) → analyze (no lock) → Completed/Failed (lock)
```

- **Creating and submitting.** `svc.CreateTask(text)` stores a Pending task. `pool.Submit(id)` marks it Queued,
  puts its ID into the channel and returns immediately; it does not wait for processing.
- **Workers.** `pool.New(store, processor, 3, 10)` starts **3 workers** (the minimum allowed is 2).
  Each worker loops over `range queue`, so it stays alive and keeps taking tasks until the queue is closed and empty.
  Only 3 goroutines ever process tasks, no matter how many tasks are submitted.
- **Queue capacity: 10.** The queue is a buffered channel of task IDs, so it can never grow.
- **Full queue → rejected.** Submit uses a non-blocking `select`. If the channel is full it returns
  `pool.ErrQueueFull` and moves the task back to Pending, so the caller can retry later. Submit never blocks,
  so no caller can be left stuck after the pool stops.

## Task lifecycle

```
Pending ──Submit──> Queued ──worker──> Processing ──ok──> Completed
   │  ▲                                    └──error──> Failed
   │  └── queue full (rollback)
   └──ProcessTask (CP1, synchronous)──> Processing
```

| From       | Operation                | To         | Otherwise |
|------------|--------------------------|------------|-----------|
| —          | `CreateTask`             | Pending    | `ErrEmptyInput` for empty/whitespace input |
| Pending    | `Submit`                 | Queued     | `ErrTaskNotFound`, `ErrInvalidTransition` (Queued/Processing/Completed/Failed), `ErrQueueFull`, `ErrPoolClosed` |
| Queued     | worker takes it          | Processing | — |
| Pending    | `ProcessTask` (sync)     | Processing | `ErrInvalidTransition` — a queued task cannot also be processed synchronously |
| Processing | processor succeeds       | Completed  | result stored on the task |
| Processing | processor fails          | Failed     | no result stored, never marked Completed |

Completed and Failed are terminal; there are no retries.

## Synchronization

| Shared data | Accessed by | Protected by |
|-------------|-------------|--------------|
| task map, `nextID`, every task's fields | callers, all workers | `Store.mu` (`sync.RWMutex`): `Get`/`All`/`Len` take a read lock, writes take a write lock |
| `Pool.closed` and sends/close on `queue` | `Submit`, `Shutdown` | `Pool.mu` (`sync.Mutex`) |
| handing a task to a worker | Submit → worker | the channel itself |
| worker termination | `Shutdown` | `sync.WaitGroup` |

- **Snapshots.** `Get`/`All`/`Create` return copies of tasks, not pointers into the map. Protecting the map
  alone would not be enough: a caller reading `t.Status` through a shared pointer would race with a worker writing it.
- **Atomic check-and-set.** `Store.Transition(id, from, to)` checks the state and writes the new one under one lock.
  Two goroutines can never both see Pending and both claim the task, so a task is processed once.
- **No lock while analyzing.** The lock is held only for the state changes, so reads and other workers
  are not blocked during processing.
- **Lock order** is always `Pool.mu → Store.mu`, and workers only take `Store.mu`, so no deadlock can occur.

## Shutdown

`pool.Shutdown()`:
1. under `Pool.mu`, sets `closed = true` and closes the channel (only once);
2. workers finish every task already in the queue (`range` drains a closed channel) and exit;
3. `wg.Wait()` returns when all workers have exited.

The pool owns the channel: only `Submit` sends to it and only `Shutdown` closes it. Because `Submit` checks `closed`
and sends under the same mutex, a send can never happen on a closed channel, even when Submit and Shutdown run at the
same time. After shutdown, `Submit` returns `ErrPoolClosed` and the task stays Pending. A second `Shutdown` is a no-op.
Accepted tasks are never discarded.

## Running

```bash
go run .               # demo
go test ./...
go test -race ./...
gofmt -l .             # prints nothing
go vet ./...
```

## Tests

- `task/store_test.go` — CP1 store tests plus atomic transitions, the sync path refusing a queued task,
  snapshot copies, and concurrent Create/Get with unique IDs.
- `analyzer/analyzer_test.go` — text statistics (CP1).
- `service/service_test.go` — synchronous lifecycle (CP1).
- `pool/pool_test.go` — pool startup and config validation; real overlap (a gated processor holds both workers inside
  `Process` at once, no `time.Sleep`); correct result per task; full queue; duplicate/invalid submissions;
  shutdown drains and rejects; Submit racing with Shutdown; concurrent Submit + Get.
