package tests

import (
	"context"
	"fmt"
	goroutiner "github.com/selyukovn/go-routiner"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_TypicalMws_PanicToError(t *testing.T) {
	ctx := context.TODO()

	// If there is no such middleware around the goroutine,
	// the whole process will be failed once panic gets to the top of the goroutine stack.
	// So can't create a test to see how things look like without it.

	mwPanicToError := goroutiner.MwPanicToError(func(panicValue any, debugStack []byte, ctx context.Context) error {
		return fmt.Errorf("MwPanicToError: %#v", panicValue)
	})

	g := func(ctx context.Context) error {
		panic("Big Panic in Little Goroutine")
	}

	expected := "MwPanicToError: \"Big Panic in Little Goroutine\""
	assert.Equal(t, expected, goroutiner.New(mwPanicToError).Batch(ctx).Add(g).Wait()[0].Error())
	assert.Equal(t, expected, goroutiner.New().Batch(ctx, mwPanicToError).Add(g).Wait()[0].Error())
	assert.Equal(t, expected, goroutiner.New().Batch(ctx).Add(g, mwPanicToError).Wait()[0].Error())
}

func Test_TypicalMws_PanicRelay(t *testing.T) {
	ctx := context.TODO()

	// `MwPanicRelay` cannot be tested without `MwPanicToError`, because of panic relay.

	mwPanicToError := goroutiner.MwPanicToError(func(panicValue any, debugStack []byte, ctx context.Context) error {
		return fmt.Errorf("MwPanicToError: %#v", panicValue)
	})

	mwPanicRelay := goroutiner.MwPanicRelay(func(panicValue any, debugStack []byte, ctx context.Context) any {
		return fmt.Sprintf("MwPanicRelay: %#v", panicValue)
	})

	g := func(ctx context.Context) error {
		panic("Big Panic in Little Goroutine")
	}

	expected := "MwPanicToError: \"MwPanicRelay: \\\"Big Panic in Little Goroutine\\\"\""
	assert.Equal(t, expected, goroutiner.New(mwPanicToError, mwPanicRelay).Batch(ctx).Add(g).Wait()[0].Error())
	assert.Equal(t, expected, goroutiner.New().Batch(ctx, mwPanicToError, mwPanicRelay).Add(g).Wait()[0].Error())
	assert.Equal(t, expected, goroutiner.New().Batch(ctx).Add(g, mwPanicToError, mwPanicRelay).Wait()[0].Error())
}
