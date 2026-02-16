package goroutiner

import "context"

type Goroutine = func(context.Context) error

type Middleware = func(Goroutine) Goroutine
