package agent

import (
	"bytes"
	"context"
	"math"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestRateLimitedReaderReadNoBufferResize(t *testing.T) {
	ctx := context.Background()
	readBuf := make([]byte, 1000)
	reader := bytes.NewReader(readBuf)
	l := rate.NewLimiter(rate.Limit(1000), math.MaxInt32) // One byte per millisecond.
	// Drain the limiter, so we can get accurate timing.
	l.WaitN(ctx, math.MaxInt32)

	r := NewRateLimitedReader(ctx, reader, l)

	writeBuf := make([]byte, 10)

	start := time.Now()
	// Since the writeBuf is only 10 bytes, we expect Read to yield
	// 10 bytes and take ~10ms.
	n, err := r.Read(writeBuf)
	if err != nil {
		t.Error("Read got err:", err)
	}
	if n != 10 {
		t.Errorf("want Read 10 bytes, got %d", n)
	}
	totalTime := time.Since(start)
	if totalTime < 10*time.Millisecond {
		t.Errorf("total time want >=10ms, got %v", totalTime)
	}
}

func TestRateLimitedReaderReadBufferResize(t *testing.T) {
	ctx := context.Background()
	readBuf := make([]byte, 2000)
	reader := bytes.NewReader(readBuf)
	l := rate.NewLimiter(rate.Limit(1000), math.MaxInt32) // One byte per millisecond.
	// Drain the limiter, so we can get accurate timing.
	l.WaitN(ctx, math.MaxInt32)

	r := NewRateLimitedReader(ctx, reader, l)
	rlr, ok := r.(rateLimitedReader)
	if !ok {
		t.Errorf("want type rateLimitedReader, got type %T", reader)
	}

	writeBuf := make([]byte, 2000)

	start := time.Now()
	// Since we've overridden the chunksPerSecond to be 100, we expect Read
	// to yield 10 bytes and take ~10ms.
	n, err := rlr.Read(writeBuf)
	if err != nil {
		t.Error("Read got err:", err)
	}
	if n != 1000 {
		t.Errorf("want Read 1000 bytes, got %d", n)
	}
	totalTime := time.Since(start)
	if totalTime < 1*time.Second {
		t.Errorf("total time want >=1s, got %v", totalTime)
	}
}
