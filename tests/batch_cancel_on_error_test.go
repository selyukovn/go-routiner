package tests

import (
	"context"
	goroutiner "github.com/selyukovn/go-routiner"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Batch_CancelOnError(t *testing.T) {
	type G = goroutiner.Goroutine
	type Mw = goroutiner.Middleware

	ctx := context.TODO()
	mw := func(g G) G { return g }
	g := func(ctx context.Context) error { return nil }

	t.Run("panic no goroutines", func(t *testing.T) {
		grt := goroutiner.New()
		assert.NotPanics(t, func() {
			_ = grt.Batch(ctx).Add(g).CancelOnError()
			_ = grt.Batch(ctx).Add(g).Add(g, mw).CancelOnError()
			_ = grt.Batch(ctx).Add(g).AddRange(1, func(i int) (G, []Mw) { return g, []Mw{mw} }).CancelOnError()
		})
		assert.Panics(t, func() { _ = grt.Batch(ctx).CancelOnError() })
	})

	// Note: all added goroutines execution is checked in the separated adding-test.

	// Note: no need to check the workflow, because `golang.org/x/sync/errgroup` package is used.
	// So can be sure all is good.
}
