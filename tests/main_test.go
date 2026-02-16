package tests

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Some test may lock goroutine forever because of unclosed channels, etc.
	go func() {
		const MaxExecutionTime = 3 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), MaxExecutionTime)
		defer cancel()
		select {
		case <-ctx.Done():
			log.Fatal("looks like something was locked forever")
		}
	}()

	os.Exit(m.Run())
}
