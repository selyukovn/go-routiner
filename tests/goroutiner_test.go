package tests

import (
	"context"
	"github.com/selyukovn/go-routiner"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Goroutiner(t *testing.T) {
	type G = goroutiner.Goroutine
	type Mw = goroutiner.Middleware
	ctx := context.TODO()
	mw := func(g G) G { return g }

	// Goroutiner.New
	// --------------------------------

	t.Run("New - panic arguments", func(t *testing.T) {
		assert.NotPanics(t, func() {
			goroutiner.New()
			goroutiner.New(nil...) // "nil" != "nil..."
			goroutiner.New([]Mw{}...)
			goroutiner.New([]Mw{mw}...)
			goroutiner.New([]Mw{mw, mw}...)
		})
		assert.Panics(t, func() { goroutiner.New(nil) }) // "nil" != "nil..."
		assert.Panics(t, func() { goroutiner.New([]Mw{nil}...) })
		assert.Panics(t, func() { goroutiner.New([]Mw{mw, nil}...) })
		assert.Panics(t, func() { goroutiner.New([]Mw{mw, mw, nil}...) })
		assert.Panics(t, func() { goroutiner.New([]Mw{mw, nil, mw}...) })
	})

	// Note: correct middleware ordering is tested in the separated middleware-ordering test.

	// Goroutiner.Batch
	// --------------------------------

	t.Run("Batch - panic arguments", func(t *testing.T) {
		assert.NotPanics(t, func() {
			goroutiner.New().Batch(ctx)
			goroutiner.New().Batch(ctx, nil...) // "nil" != "nil..."
			goroutiner.New().Batch(ctx, []Mw{}...)
			goroutiner.New().Batch(ctx, []Mw{mw}...)
			goroutiner.New().Batch(ctx, []Mw{mw, mw}...)
		})
		assert.Panics(t, func() { goroutiner.New().Batch(nil) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx, nil) }) // "nil" != "nil..."
		assert.Panics(t, func() { goroutiner.New().Batch(ctx, []Mw{nil}...) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx, []Mw{mw, nil}...) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx, []Mw{mw, mw, nil}...) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx, []Mw{mw, nil, mw}...) })
	})

	// Note: correct middleware ordering is tested in the separated middleware-ordering test.

	t.Run("Batch - same global and different batch middleware", func(t *testing.T) {
		var actual string

		mMw := func(name string) Mw {
			return func(g G) G {
				return func(ctx context.Context) error {
					actual += name + "s-"
					err := g(ctx)
					actual += "-" + name + "e"
					return err
				}
			}
		}

		g := func(ctx context.Context) error {
			actual += "0"
			return nil
		}

		grt := goroutiner.New(mMw("g1"), mMw("g2"))

		// test case 1 -- different batch, but same test-result

		s1 := grt.Batch(ctx, mMw("s1"))
		s11 := grt.Batch(ctx, mMw("s1"))
		assert.NotEqual(t, s1, s11)

		actual = ""
		s1.Add(g).Wait()
		actual1 := actual

		actual = ""
		s11.Add(g).Wait()
		actual11 := actual

		assert.Equal(t, actual1, actual11)

		// test case 2 -- different batch and different test-result

		s2 := grt.Batch(ctx, mMw("s2"))
		s22 := grt.Batch(ctx, mMw("s22"))
		assert.NotEqual(t, s2, s22)

		actual = ""
		s2.Add(g).Wait()
		actual2 := actual

		actual = ""
		s22.Add(g).Wait()
		actual22 := actual

		assert.NotEqual(t, actual2, actual22)
	})
}
