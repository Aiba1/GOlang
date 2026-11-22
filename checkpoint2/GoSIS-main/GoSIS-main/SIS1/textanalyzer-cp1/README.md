# Text Analyzer — SIS Checkpoint 1

Individual Checkpoint 1 project for **Golang Application Development**.
A small, synchronous, in-memory Go application built around the concept
of a **task**: here, a task is a request to analyze a piece of text and
produce word/character statistics for it.

## Domain

**Text Analyzer.** A task's input is a string of raw text. Processing
computes:

- word count
- character count (total runes, including whitespace)
- character count excluding whitespace
- the longest word
- average word length

Example (matches the spec's own sample):

```
Input:  "Hello Go world"
Result: words=3 chars=14 chars_no_spaces=12 longest_word="Hello" avg_word_length=4.00
```

## Package layout

```
task/           domain model — Task, Status, the Result and Processor
                interfaces, and the in-memory Store. Knows nothing about
                how text is analyzed.
  task.go       Task, Status, Result interface, Processor interface
  store.go      in-memory Store: Create / Get / All / Len
  errors.go     sentinel errors (ErrEmptyInput, ErrTaskNotFound, ErrInvalidTransition)

analyzer/       the concrete processing strategy for this domain.
  analyzer.go   TextAnalyzer (implements task.Processor) + Result (implements task.Result)

service/        application-level orchestration.
  service.go    TaskService: CreateTask / GetTask / ProcessTask — wires
                a task.Store to any task.Processor and enforces the
                lifecycle transitions below.

main.go         small manual demo (go run .)
```

**Why split it this way:** `task` defines *what* a result and a
processor look like (as interfaces) without knowing *how* analysis is
done. `analyzer` provides one concrete way to do it. `service` depends
only on the `task.Processor` interface, not on `analyzer` directly —
so the tests in `service/service_test.go` use a small fake processor
instead of the real text analyzer, and a different processor could be
substituted later without touching `service.go`.

## Task lifecycle

```
Pending ──ProcessTask()──> Processing ──(success)──> Completed
                                 │
                                 └────(processor error)──> Failed
```

| From       | Operation         | To         | Notes                                   |
|------------|-------------------|------------|------------------------------------------|
| —          | `CreateTask`      | Pending    | rejected with `ErrEmptyInput` if input is empty/whitespace-only |
| Pending    | `ProcessTask`     | Processing | then immediately Completed or Failed, since processing is synchronous |
| Processing | (processor succeeds) | Completed | result is attached to the task |
| Processing | (processor errors)   | Failed    | no result is attached |
| Completed  | `ProcessTask`     | — rejected | returns `ErrInvalidTransition`; a completed task cannot be reprocessed |
| Failed     | `ProcessTask`     | — rejected | returns `ErrInvalidTransition`; same reasoning as above |

`Completed` and `Failed` are both terminal states for this checkpoint.
`Failed` is reachable in principle (any `task.Processor` can return an
error), even though the bundled `TextAnalyzer` cannot actually produce
one once `Store.Create` has already validated its input.

## Running

```bash
go run .              # demo: creates/processes a few tasks, prints the lifecycle
go test ./...         # all tests
gofmt -l .             # should print nothing
go vet ./...
```

## Tests

- `task/store_test.go` — task creation (success + table-driven invalid
  input), retrieval (success + not-found), and that multiple tasks
  stay independent (no shared/overwritten state).
- `analyzer/analyzer_test.go` — table-driven correctness of the word/
  character statistics, plus invalid-input handling.
- `service/service_test.go` — the lifecycle: successful processing,
  processing a missing task, reprocessing a completed task (invalid
  transition), a processor failure path, and that processing one task
  doesn't affect another.

## Notes on the spec

The spec asks for **Go 1.27**; the toolchain available in this
environment is **Go 1.24.7**, which is what `go.mod` currently
declares (`go 1.24`). Nothing in this code uses any version-specific
language feature, so bumping the `go` directive once 1.27 is actually
installed should be a one-line change — but it's worth checking which
Go version your grading environment actually has before submitting,
in case the spec means something that hasn't been released yet.
