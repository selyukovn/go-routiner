package tests

import (
	"context"
	goroutiner "github.com/selyukovn/go-routiner"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Batch_Adding(t *testing.T) {
	type G = goroutiner.Goroutine
	type Mw = goroutiner.Middleware

	ctx := context.TODO()

	t.Run("panic arguments", func(t *testing.T) {
		mw := func(g G) G { return g }
		g := func(ctx context.Context) error { return nil }

		// Add
		// ----------------

		assert.NotPanics(t, func() {
			goroutiner.New().Batch(ctx).Add(g)
			goroutiner.New().Batch(ctx).Add(g, nil...) // "nil" != "nil..."
			goroutiner.New().Batch(ctx).Add(g, []Mw{}...)
			goroutiner.New().Batch(ctx).Add(g, []Mw{mw}...)
			goroutiner.New().Batch(ctx).Add(g, []Mw{mw, mw}...)
		})
		assert.Panics(t, func() { goroutiner.New().Batch(ctx).Add(nil) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx).Add(g, nil) }) // "nil" != "nil..."
		assert.Panics(t, func() { goroutiner.New().Batch(ctx).Add(g, []Mw{nil}...) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx).Add(g, []Mw{mw, nil}...) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx).Add(g, []Mw{mw, mw, nil}...) })
		assert.Panics(t, func() { goroutiner.New().Batch(ctx).Add(g, []Mw{mw, nil, mw}...) })

		// AddRange
		// ----------------

		assert.NotPanics(t, func() {
			b := goroutiner.New().Batch(ctx)
			b.AddRange(1, func(i int) (G, []Mw) { return g, nil })
			b.AddRange(2, func(i int) (G, []Mw) { return g, []Mw{} })
			b.AddRange(3, func(i int) (G, []Mw) { return g, []Mw{mw} })
			b.AddRange(4, func(i int) (G, []Mw) { return g, []Mw{mw, mw} })
		})
		b := goroutiner.New().Batch(ctx)
		assert.Panics(t, func() { b.AddRange(-1, func(i int) (G, []Mw) { return g, nil }) })
		assert.Panics(t, func() { b.AddRange(0, func(i int) (G, []Mw) { return g, nil }) })
		assert.Panics(t, func() { b.AddRange(1, nil) })
		assert.Panics(t, func() { b.AddRange(2, func(i int) (G, []Mw) { return nil, []Mw{mw} }) })
		assert.Panics(t, func() { b.AddRange(3, func(i int) (G, []Mw) { return g, []Mw{nil} }) })
		assert.Panics(t, func() { b.AddRange(4, func(i int) (G, []Mw) { return g, []Mw{mw, nil} }) })
		assert.Panics(t, func() { b.AddRange(5, func(i int) (G, []Mw) { return g, []Mw{mw, nil, mw} }) })

		// be sure, that the panic-case range is not half-added,
		// (e.g. we don't have 3/5 goroutines added to the batch before panic occurred on the 4-th element)
		// i.e. all result methods must panic because of no goroutines added.
		assert.Panics(t, func() {
			b.AddRange(5, func(i int) (G, []Mw) {
				if i < 4 {
					return g, nil
				} else {
					return g, []Mw{mw, nil, mw}
				}
			})
		})
		assert.Panics(t, func() { _ = b.Wait() })
		assert.Panics(t, func() { _ = b.CancelOnError() })
		assert.Panics(t, func() { _ = b.Async() })
		assert.Panics(t, func() { _ = b.AsyncBs(5) })
	})

	// Note: correct middleware ordering is tested in the separated middleware-ordering test.

	t.Run("all added executed", func(t *testing.T) {
		// Checks, that all added goroutines are executed,
		// and individual middleware wraps only their goroutine.

		const rs = 10
		type Res [rs]int64 // 64 -- because of 9999999999 test case

		var actual Res

		mMw := func(i int, n int64) Mw {
			return func(next G) G {
				return func(ctx context.Context) error {
					actual[i] += n
					return next(ctx)
				}
			}
		}

		mG := func(i int, n int64) G {
			return func(ctx context.Context) error {
				actual[i] += n
				return nil
			}
		}

		// ----

		type TGCnf *struct {
			fn  G
			mws []Mw
		}
		tCases := []struct {
			gConfigs [rs]TGCnf
			expected Res
		}{
			// nil always for 0 element to start results from 1 and make nice results like x + x * 10 + x * 100 + ...
			{[rs]TGCnf{nil, {mG(1, 1), nil}, {mG(2, 2), nil}, {mG(3, 3), nil}}, Res{0, 1, 2, 3}},
			{[rs]TGCnf{nil, {mG(1, 1), []Mw{mMw(1, 10)}}, nil, {mG(3, 3), []Mw{mMw(3, 30)}}}, Res{0, 11, 0, 33}},
			{[rs]TGCnf{nil, nil, {mG(2, 2), []Mw{mMw(2, 200), mMw(2, 20)}}, {mG(3, 3), []Mw{mMw(3, 30)}}}, Res{0, 0, 222, 33}},
			{
				[rs]TGCnf{
					nil,
					{mG(1, 1), []Mw{mMw(1, 10)}},
					{mG(2, 2), []Mw{mMw(2, 200), mMw(2, 20)}},
					{mG(3, 3), []Mw{mMw(3, 3000), mMw(3, 300), mMw(3, 30)}},
					{mG(4, 4), []Mw{mMw(4, 40000), mMw(4, 4000), mMw(4, 400), mMw(4, 40)}},
					{mG(5, 5), []Mw{mMw(5, 500000), mMw(5, 50000), mMw(5, 5000), mMw(5, 500), mMw(5, 50)}},
					{mG(6, 6), []Mw{mMw(6, 6000000), mMw(6, 600000), mMw(6, 60000), mMw(6, 6000), mMw(6, 600), mMw(6, 60)}},
					{mG(7, 7), []Mw{mMw(7, 70000000), mMw(7, 7000000), mMw(7, 700000), mMw(7, 70000), mMw(7, 7000), mMw(7, 700), mMw(7, 70)}},
					{mG(8, 8), []Mw{mMw(8, 800000000), mMw(8, 80000000), mMw(8, 8000000), mMw(8, 800000), mMw(8, 80000), mMw(8, 8000), mMw(8, 800), mMw(8, 80)}},
					{mG(9, 9), []Mw{mMw(9, 9000000000), mMw(9, 900000000), mMw(9, 90000000), mMw(9, 9000000), mMw(9, 900000), mMw(9, 90000), mMw(9, 9000), mMw(9, 900), mMw(9, 90)}},
				},
				Res{0, 11, 222, 3333, 44444, 555555, 6666666, 77777777, 888888888, 9999999999},
			},
		}

		testBatch := func(batch *goroutiner.Batch, tCaseI int, expected Res) {
			// perhaps, no need to use all the result methods -- but why not?

			// wg
			actual = Res{}
			batch.Wait()
			assert.Equal(t, expected, actual, "tCase %d - %s.Wait", tCaseI)

			// err-group
			actual = Res{}
			_ = batch.CancelOnError()
			assert.Equal(t, expected, actual, "tCase %d - %s.CancelOnError", tCaseI)

			// async
			actual = Res{}
			for err := range batch.Async() {
				_ = err
			}
			assert.Equal(t, expected, actual, "tCase %d - %s.Async", tCaseI)

			// async-bs
			for bs := 0; bs < rs; bs++ {
				actual = Res{}
				for err := range batch.AsyncBs(uint(bs)) {
					_ = err
				}
				assert.Equal(t, expected, actual, "tCase %d - %s.AsyncBs-%d", tCaseI, "Add", bs)
			}
		}

		// ----

		grt := goroutiner.New()

		for tci, tCase := range tCases {
			gConfigs := make([]TGCnf, 0)
			for _, gCnf := range tCase.gConfigs {
				if gCnf != nil {
					gConfigs = append(gConfigs, gCnf)
				}
			}

			// Add
			// ----------------

			batchAdd := grt.Batch(ctx)
			for _, gCnf := range gConfigs {
				batchAdd.Add(gCnf.fn, gCnf.mws...)
			}
			testBatch(batchAdd, tci, tCase.expected)

			// AddRange
			// ----------------

			batchAddRange := grt.Batch(ctx).AddRange(len(gConfigs), func(i int) (G, []Mw) {
				return gConfigs[i].fn, gConfigs[i].mws
			})
			testBatch(batchAddRange, tci, tCase.expected)
		}
	})
}
