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

func Test_Batch_Async(t *testing.T) {
	type G = goroutiner.Goroutine
	type Mw = goroutiner.Middleware

	ctx := context.TODO()
	t.Run("panic no goroutines", func(t *testing.T) {
		mw := func(g G) G { return g }
		g := func(ctx context.Context) error { return nil }

		assert.NotPanics(t, func() {
			_ = goroutiner.New().Batch(ctx).Add(g).Async()
			_ = goroutiner.New().Batch(ctx).Add(g).Add(g, mw).Async()
			_ = goroutiner.New().Batch(ctx).Add(g).AddRange(1, func(i int) (G, []Mw) { return g, []Mw{mw} }).Async()
		})
		assert.Panics(t, func() { _ = goroutiner.New().Batch(ctx).Async() })
	})

	// Note: all added goroutines execution is checked in the separated adding-test.

	t.Run("result errors", func(t *testing.T) {
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
			errCh := b.Async()

			// reading here locks the flow until the channel is closed -- i.e. execution flow is synced.
			resultItemsTotal := 0
			errsCountMap := make(map[string]int)
			for err := range errCh {
				resultItemsTotal++
				if err == nil {
					if _, ok := errsCountMap[""]; !ok {
						errsCountMap[""] = 0
					}
					errsCountMap[""]++
				} else {
					if _, ok := errsCountMap[err.Error()]; !ok {
						errsCountMap[err.Error()] = 0
					}
					errsCountMap[err.Error()]++
				}
			}

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
			assert.Equal(t, len(tCase.gs), resultItemsTotal, "tCase #%d: result items count != num of goroutines", tci)

			// indexes are random, so we can check only errors existence, not the order.
			for _, expectedErrMsg := range tCase.expected {
				_, exists := errsCountMap[expectedErrMsg]
				assert.True(t, exists, "tCase #%d: %q must be in result errors", tci, expectedErrMsg)
				errsCountMap[expectedErrMsg]--
			}
			for k := range errsCountMap {
				assert.Equal(t, 0, errsCountMap[k], "tCase #%d: %q mismatch", tci, k)
			}
		}
	})

	t.Run("channel is closed", func(t *testing.T) {
		grt := goroutiner.New()
		for numOfGoroutines := 1; numOfGoroutines <= 5; numOfGoroutines++ {
			// func() -- for defer ctxCancel() -- i.e. to be sure on each iteration.
			func() {
				ctx, ctxCancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer ctxCancel()

				errCh := grt.
					Batch(ctx).
					AddRange(numOfGoroutines, func(i int) (G, []Mw) {
						return func(ctx context.Context) error {
							return fmt.Errorf("error #%d", i)
						}, nil
					}).
					Async()

				// `numOfGoroutines + 1` read must say, that channel is closed.
				for i := 1; i <= numOfGoroutines+1; i++ {
					select {
					case _, isOpen := <-errCh:
						if i == numOfGoroutines+1 {
							// All errors are read already -- so channel must be closed.
							// Otherwise, there will be endless waiting for a new elements here.
							assert.False(t, isOpen)
						} else {
							assert.True(t, isOpen)
						}
					case <-ctx.Done():
						assert.True(t, false, "tCase #%d: exit by timeout -- looks like channel was not closed", i)
					}
				}
			}()
		}
	})

	t.Run("really async", func(t *testing.T) {
		deadline := time.Now().Add(50 * time.Millisecond)
		ctx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		_ = goroutiner.New().Batch(ctx).Add(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			}
		}).Async()

		assert.True(t, time.Now().Before(deadline))
	})

	t.Run("buffer size", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			errCh := goroutiner.New().Batch(ctx).Add(func(ctx context.Context) error { return nil }).AsyncBs(uint(i))
			assert.Equal(t, i, cap(errCh))
		}
	})
}
