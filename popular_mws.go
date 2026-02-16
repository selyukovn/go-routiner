package goroutiner

import (
	"context"
	"runtime/debug"
)

// MwPanicRelay creates a middleware that intercepts panics, passes the panic value to `fnHandler`,
// then re‑panics using the value returned by `fnHandler`.
//
// Example use case: log or transform the panic data before propagating the panic.
//
// Panics if `fnHandler` is nil.
func MwPanicRelay(fnHandler func(panicValue any, debugStack []byte, ctx context.Context) any) Middleware {
	if fnHandler == nil {
		panic("`fnHandler` must not be `nil`")
	}

	return func(g Goroutine) Goroutine {
		return func(ctx context.Context) error {
			defer func() {
				if pv := recover(); pv != nil {
					pv = fnHandler(pv, debug.Stack(), ctx)
					panic(pv)
				}
			}()

			return g(ctx)
		}
	}
}

// MwPanicToError creates a middleware that intercepts panics and converts the panic value into an error
// using `fnHandler`, instead of letting the panic propagate.
//
// Example use case: graceful error handling in long‑running services.
//
// Panics if `fnHandler` is nil.
func MwPanicToError(fnHandler func(panicValue any, debugStack []byte, ctx context.Context) error) Middleware {
	if fnHandler == nil {
		panic("`fnHandler` must not be `nil`")
	}

	return func(g Goroutine) Goroutine {
		return func(ctx context.Context) (rErr error) {
			defer func() {
				if pv := recover(); pv != nil {
					rErr = fnHandler(pv, debug.Stack(), ctx)
				}
			}()

			rErr = g(ctx)

			return
		}
	}
}
