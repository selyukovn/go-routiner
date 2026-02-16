package goroutiner

import (
	"context"
	"golang.org/x/sync/errgroup"
	"sync"
)

// ---------------------------------------------------------------------------------------------------------------------
// Struct
// ---------------------------------------------------------------------------------------------------------------------

// Batch
//
// Not thread-safe, as there is no practical need to make it thread‑safe.
type Batch struct {
	ctx              context.Context
	mws              []Middleware
	goroutineConfigs []*goroutineConfig
}

type goroutineConfig struct {
	fn  Goroutine
	mws []Middleware
}

// ---------------------------------------------------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------------------------------------------------

func newBatch(ctx context.Context, mws []Middleware) *Batch {
	return &Batch{
		ctx:              ctx,
		mws:              mws,
		goroutineConfigs: make([]*goroutineConfig, 0),
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------------------------------------------------

// Add adds a new goroutine to the Batch, with optional individual middleware.
// Individual middleware will be applied only to currently added goroutine after the most inner batch middleware.
// Middleware order: first = outermost.
//
// Panics if `fn` is nil.
// Panics if any element in `mws` is nil.
func (b *Batch) Add(fn Goroutine, mws ...Middleware) *Batch {
	if fn == nil {
		panic("`fn` must not be `nil`")
	}

	for _, mw := range mws {
		if mw == nil {
			panic("`mws` must contain no `nil` elements")
		}
	}

	b.goroutineConfigs = append(b.goroutineConfigs, &goroutineConfig{
		fn:  fn,
		mws: mws,
	})

	return b
}

// AddRange adds `n` goroutines using the `fnProvide`.
// If a panic occurs, none of the provided goroutines are added.
//
// Panics if:
//   - `n` <= 0
//   - `fnProvide` is nil
//   - `fnProvide` returns nil goroutine
//   - `fnProvide` returns middleware containing `nil`
func (b *Batch) AddRange(n int, fnProvide func(i int) (Goroutine, []Middleware)) *Batch {
	if n <= 0 {
		panic("`n` must be greater than zero")
	}

	if fnProvide == nil {
		panic("`fnProvide` must not be `nil`")
	}

	// Collects all provided goroutines before adding
	// to avoid half-added set of goroutines (e.g. in case of nil-element panic).
	// I.e. to be sure that "all or nothing" is added.
	configs := make([]*goroutineConfig, n)

	for i := 0; i < n; i++ {
		fn, mws := fnProvide(i)

		if fn == nil {
			panic("`fnProvide` must not provide `nil` goroutines")
		}

		for _, mw := range mws {
			if mw == nil {
				panic("`fnProvide` must provide middleware without `nil` elements")
			}
		}

		configs[i] = &goroutineConfig{
			fn:  fn,
			mws: mws,
		}
	}

	b.goroutineConfigs = append(b.goroutineConfigs, configs...)

	return b
}

// Executing
// ---------------------------------------------------------------------------------------------------------------------

func (b *Batch) prepareGoroutines() []Goroutine {
	goroutines := make([]Goroutine, len(b.goroutineConfigs))

	if len(goroutines) == 0 {
		panic("at least one goroutine is required")
	}

	for i, cfg := range b.goroutineConfigs {
		goroutines[i] = cfg.fn

		for j := len(cfg.mws) - 1; j >= 0; j-- {
			goroutines[i] = cfg.mws[j](goroutines[i])
		}

		for j := len(b.mws) - 1; j >= 0; j-- {
			goroutines[i] = b.mws[j](goroutines[i])
		}
	}

	return goroutines
}

// Execution - Wait
// ---------------------------------------------------------------------------------------------------------------------

// Wait executes all goroutines using sync.WaitGroup: waits for completion and collects all returned errors.
// Returns a slice of errors: index `i` matches `i`-th added goroutine.
//
// Panics if no goroutines were added to the Batch.
func (b *Batch) Wait() []error {
	gs := b.prepareGoroutines()

	type ChErr = struct {
		I   int
		Err error
	}

	errCh := make(chan ChErr, len(gs))

	// wrapped into a function to be sure, that channel will be closed in any case after wg.Wait()
	// so reading from the channel will not lock the goroutine after all elements are pulled.
	func(errCh chan ChErr, gs []Goroutine, ctx context.Context) {
		defer close(errCh)

		wg := new(sync.WaitGroup)
		wg.Add(len(gs))
		for i, g := range gs {
			go func(i int, g func(context.Context) error) {
				defer wg.Done()

				err := g(ctx)

				errCh <- ChErr{i, err}
			}(i, g)
		}
		wg.Wait()
	}(errCh, gs, b.ctx)

	errs := make([]error, len(gs))
	for chErr := range errCh {
		errs[chErr.I] = chErr.Err
	}

	return errs
}

// Execution - CancelOnError
// ---------------------------------------------------------------------------------------------------------------------

// CancelOnError executes all goroutines using errgroup.WithContext -- see https://pkg.go.dev/golang.org/x/sync/errgroup.
//
// "...
// WithContext returns a new Group and an associated Context derived from ctx.
// The derived Context is canceled the first time a function passed to Go returns a non-nil error
// or the first time Wait returns, whichever occurs first.
// ..."
//
// Panics if no goroutines were added to the Batch.
func (b *Batch) CancelOnError() error {
	gs := b.prepareGoroutines()

	eg, egCtx := errgroup.WithContext(b.ctx)

	for _, g := range gs {
		func(g Goroutine) {
			eg.Go(func() error {
				return g(egCtx)
			})
		}(g)
	}

	return eg.Wait()
}

// Execution - Async
// ---------------------------------------------------------------------------------------------------------------------

func (b *Batch) async(gs []Goroutine, errChBufferSize uint) <-chan error {
	errCh := make(chan error, errChBufferSize)

	go func(errCh chan<- error, gs []Goroutine, ctx context.Context) {
		defer close(errCh)

		wg := new(sync.WaitGroup)
		wg.Add(len(gs))

		for _, g := range gs {
			go func(g Goroutine) {
				defer wg.Done()
				errCh <- g(ctx)
			}(g)
		}

		wg.Wait()
	}(errCh, gs, b.ctx)

	return errCh
}

// Async executes all goroutines asynchronously, i.e. without awaiting goroutines completion.
// Returns a buffered channel (with capacity equal to the number of added goroutines) for receiving errors from the goroutines.
// Channel will be closed after all goroutines are executed.
//
// Panics if no goroutines were added to the Batch.
func (b *Batch) Async() <-chan error {
	gs := b.prepareGoroutines()

	return b.async(gs, uint(len(gs)))
}

// AsyncBs is the same as Async, but with custom buffer size of the result channel.
//
// Panics if no goroutines were added to the Batch.
func (b *Batch) AsyncBs(errChBufferSize uint) <-chan error {
	gs := b.prepareGoroutines()

	return b.async(gs, errChBufferSize)
}

// ---------------------------------------------------------------------------------------------------------------------
