package goroutiner

import (
	"context"
)

// ---------------------------------------------------------------------------------------------------------------------
// Struct
// ---------------------------------------------------------------------------------------------------------------------

type Goroutiner struct {
	globalMws []Middleware
}

// ---------------------------------------------------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------------------------------------------------

// New
//
// Global middleware will be applied to every Goroutine in every Batch.
// Middleware order: first = outermost.
//
// Panics if:
//   - `globalMws` contains nil.
func New(globalMws ...Middleware) *Goroutiner {
	for _, mw := range globalMws {
		if mw == nil {
			panic("`globalMws` must contain no `nil` elements")
		}
	}

	return &Goroutiner{
		globalMws: globalMws,
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------------------------------------------------

// Batch creates a new batch with the given context and optional batch middleware.
// The context will be passed to all added goroutines.
// Batch middleware are applied to each goroutine after the innermost global middleware.
// Middleware order: first = outermost.
//
// Panics if:
//   - `ctx` is nil
//   - `mws` contains nil
func (g *Goroutiner) Batch(ctx context.Context, mws ...Middleware) *Batch {
	if ctx == nil {
		panic("`ctx` must not be `nil`")
	}

	for _, mw := range mws {
		if mw == nil {
			panic("`mws` must contain no `nil` elements")
		}
	}

	batchMws := make([]Middleware, 0, len(g.globalMws)+len(mws))
	batchMws = append(batchMws, g.globalMws...)
	batchMws = append(batchMws, mws...)

	return newBatch(ctx, batchMws)
}

// SingleAsync -- alias to Batch.Async() with only one added goroutine.
func (g *Goroutiner) SingleAsync(ctx context.Context, fn Goroutine, mws ...Middleware) <-chan error {
	return g.Batch(ctx).Add(fn, mws...).Async()
}

// ---------------------------------------------------------------------------------------------------------------------
