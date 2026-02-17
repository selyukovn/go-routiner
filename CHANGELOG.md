## [0.1.0] - 2026-02-17

First implementation of the package.

### REQUIREMENTS

- Go 1.18+ (no sense in earlier versions).

### FEATURES

- Fluent chaining (e.g., `goroutiner.New().Batch(ctx).Add(go1).Add(go2).Wait()`)

- 2 ways of collecting goroutines for execution:
    - `Add()` -- for single goroutine
    - `AddRange()` -- for multiple goroutines

- 3 sets of goroutine middleware: `global`, `batch`, `individual`.

- 2 methods to create typical middleware for panic handling:
    - `MwPanicToError()` -- converts a panic to an error, can be used to prevent the app from crashing
    - `MwPanicRelay()` -- allows custom handling of panics before they propagate further (e.g., for logging purposes)

- 3 typical execution strategies:
    - `Wait()` -- wraps the `sync.WaitGroup` mechanism, returns collected errors from goroutines.
    - `CancelOnError()` -- wraps the [`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup)
      mechanism.
    - `Async()` / `AsyncBs()` -- executes goroutines asynchronously and returns a channel for handling errors.

- Alias `SingleAsync()` for single goroutine execution via `Async()` strategy.

- Tests for:
    - `Goroutiner` instance creation and making `Batch`es
    - collecting goroutines within a `Batch`
    - middleware combination and application order
    - `Wait()` and `Async()` execution strategies
