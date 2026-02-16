package tests

import (
	"context"
	"github.com/selyukovn/go-routiner"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_MiddlewareOrdering(t *testing.T) {
	type G = goroutiner.Goroutine
	type Mw = goroutiner.Middleware

	type TCase = struct {
		name     string
		gMws     []Mw
		sMws     []Mw
		iMws     []Mw
		expected string
	}

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

	ctx := context.TODO()

	for _, tCase := range []TCase{
		{"gg", []Mw{mMw("g1"), mMw("g2")}, nil, nil, "g1s-g2s-0-g2e-g1e"},
		{"ss", nil, []Mw{mMw("s1"), mMw("s2")}, nil, "s1s-s2s-0-s2e-s1e"},
		{"ii", nil, nil, []Mw{mMw("i1"), mMw("i2")}, "i1s-i2s-0-i2e-i1e"},
		{"ggssii", []Mw{mMw("g1"), mMw("g2")}, []Mw{mMw("s1"), mMw("s2")}, []Mw{mMw("i1"), mMw("i2")}, "g1s-g2s-s1s-s2s-i1s-i2s-0-i2e-i1e-s2e-s1e-g2e-g1e"},
		{"gs", []Mw{mMw("g1")}, []Mw{mMw("s1")}, nil, "g1s-s1s-0-s1e-g1e"},
		{"gi", []Mw{mMw("g1")}, nil, []Mw{mMw("i1")}, "g1s-i1s-0-i1e-g1e"},
		{"si", nil, []Mw{mMw("s1")}, []Mw{mMw("i1")}, "s1s-i1s-0-i1e-s1e"},
	} {
		// Add
		actual = ""
		_ = goroutiner.New(tCase.gMws...).
			Batch(ctx, tCase.sMws...).
			Add(g, tCase.iMws...).
			Wait()
		assert.Equal(t, tCase.expected, actual, "tCase: %s", tCase.name)

		// AddRange
		actual = ""
		_ = goroutiner.New(tCase.gMws...).
			Batch(ctx, tCase.sMws...).
			AddRange(1, func(i int) (G, []Mw) { return g, tCase.iMws }).
			Wait()
		assert.Equal(t, tCase.expected, actual, "tCase: %s", tCase.name)
	}
}
