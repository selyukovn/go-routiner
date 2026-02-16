package tests

import (
	"context"
	"fmt"
	goroutiner "github.com/selyukovn/go-routiner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_Batch_Wait(t *testing.T) {
	type G = goroutiner.Goroutine
	type Mw = goroutiner.Middleware

	ctx := context.TODO()

	t.Run("panic no goroutines", func(t *testing.T) {
		mw := func(g G) G { return g }
		g := func(ctx context.Context) error { return nil }

		assert.NotPanics(t, func() {
			_ = goroutiner.New().Batch(ctx).Add(g).Wait()
			_ = goroutiner.New().Batch(ctx).Add(g).Add(g, mw).Wait()
			_ = goroutiner.New().Batch(ctx).Add(g).AddRange(2, func(i int) (G, []Mw) { return g, []Mw{mw} }).Wait()
		})
		assert.Panics(t, func() { _ = goroutiner.New().Batch(ctx).Wait() })
	})

	// Note: all added goroutines execution is checked in the separated adding-test.

	t.Run("result errors [i] = g[i] error", func(t *testing.T) {
		const maxG = 10
		type GoroutinesRunFlags = [maxG]int

		actualGoroutinesRunFlags := GoroutinesRunFlags{}

		mG := func(i int, errMsg string) G {
			return func(ctx context.Context) error {
				if errMsg == "" {
					actualGoroutinesRunFlags[i] = -1
					return nil
				} else {
					actualGoroutinesRunFlags[i] = 1
					return fmt.Errorf(errMsg)
				}
			}
		}

		tCases := []struct {
			gs       []G
			expected []string
		}{
			{[]G{mG(0, "")}, []string{""}},
			{[]G{mG(0, ""), mG(1, ""), mG(2, "")}, []string{"", "", ""}},
			{[]G{mG(0, "a")}, []string{"a"}},
			{[]G{mG(0, "a"), mG(1, "b")}, []string{"a", "b"}},
			{[]G{mG(0, "a"), mG(1, ""), mG(2, "c"), mG(3, ""), mG(4, "e")}, []string{"a", "", "c", "", "e"}},
		}

		grt := goroutiner.New()
		for tci, tCase := range tCases {
			actualGoroutinesRunFlags = GoroutinesRunFlags{}
			b := grt.Batch(ctx)
			for _, g := range tCase.gs {
				b.Add(g)
			}
			wgErrs := b.Wait()

			// ----------------------------------------------------------------
			// Check test configuration:
			// first len(tCase.gs) GoroutinesRunFlags must be != 0, and all others = 0
			for i, f := range actualGoroutinesRunFlags {
				if i < len(tCase.gs) {
					require.NotEqual(t, 0, f, "tCase #%d/%d: incorrect test configuration: %d", tci, i, f)
				} else {
					require.Equal(t, 0, f, "tCase #%d/%d: incorrect test configuration: %d", tci, i, f)
				}
			}
			// ----------------------------------------------------------------

			// result length = number of goroutines
			assert.Equal(t, len(tCase.gs), len(wgErrs))

			tCaseActual := make([]string, len(tCase.gs))
			for i, err := range wgErrs {
				if err == nil {
					tCaseActual[i] = ""
					assert.Equal(t, -1, actualGoroutinesRunFlags[i])
				} else {
					tCaseActual[i] = err.Error()
					assert.Equal(t, 1, actualGoroutinesRunFlags[i])
				}
			}
			// same indexes
			assert.Equal(t, tCase.expected, tCaseActual)
		}
	})

	t.Run("not async", func(t *testing.T) {
		deadline := time.Now().Add(50 * time.Millisecond)
		ctx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		_ = goroutiner.New().Batch(ctx).Add(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			}
		}).Wait()

		assert.True(t, time.Now().After(deadline))
	})
}
