# CP2 — concurrent Text Analyzer (worker pool) — design

**Intent.** Extend the CP1 Text Analyzer (`SIS2/`, module `textanalyzer-cp1`) per `message.txt`:
async submission into a bounded queue, fixed worker pool, safe shared state, clean shutdown.
Must stay small enough to defend line-by-line in 2 minutes.

**Non-goals.** context cancellation, HTTP, retries, blocking submit, metrics.

**Approaches.** Only one sensible way given the spec: buffered channel + N workers + mutex-guarded store.
Considered: blocking Submit (rejected — needs extra unblock-on-shutdown logic); per-task mutex
(rejected — one store-level RWMutex is simpler and enough); channel-owned store goroutine (rejected — harder to explain).

**Design.**
- `task.Store`: `sync.RWMutex` guards map + nextID + task fields. `Get`/`All` return copies (`Task` values),
  never shared pointers. `Transition(id, from, to)` = atomic check-and-set under lock.
  `Process(id, from, processor)` = from→Processing, analyze without lock, then Completed/Failed (one path for sync + workers).
- States: `Pending → Queued → Processing → Completed | Failed`. Only Pending may be submitted.
- `pool.Pool`: `New(store, processor, workers, queueCap)` (starts workers), `Submit(id)`, `Shutdown()`.
  Queue = `chan string` (task IDs), capacity = queueCap. Defaults in main: 3 workers, capacity 10.
- Submit: under pool mutex: closed? → `ErrPoolClosed`; Pending→Queued; non-blocking send;
  full → revert Queued→Pending, `ErrQueueFull`. Mutex shared with Shutdown ⇒ no send on closed channel.
- Worker: `for id := range queue { store.Process(id, Queued, proc) }`.
- Shutdown: under mutex set closed + close(queue) once; then `wg.Wait()`. Second call = no-op.
  Queued tasks are drained, not discarded.
- `service.TaskService.ProcessTask` (sync, CP1) kept, = `store.Process(id, Pending, proc)`.
- Tests: pool startup, submit+results, overlap proof (blocking fake processor waits until 2 in flight),
  duplicate/invalid submit, queue full (state stays Pending), concurrent Get, shutdown drains, submit after shutdown,
  double shutdown. All pass `go test -race`.
- README updated; go.mod → go 1.27.

**Left to builder.** naming, exact test structure.
