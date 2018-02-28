package agent

import (
	"bytes"
	"context"
	"io"
	"math"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRateLimitedReader(t *testing.T) {
	ctx := context.Background()
	buf := make([]byte, 10)
	r := bytes.NewReader(buf)
	var testCases = []struct {
		ctx            context.Context
		reader         io.Reader
		bytesPerSecond rate.Limit
		wantLimit      rate.Limit
		wantBurst      int
	}{
		{ctx, r, 100, 100, math.MaxInt32},
		{ctx, r, 0, rate.Inf, math.MaxInt32},
		{ctx, r, -1, rate.Inf, math.MaxInt32},
		{ctx, r, math.MaxFloat64, math.MaxFloat64, math.MaxInt32},
	}
	for _, tc := range testCases {
		reader, err := NewRateLimitedReader(tc.ctx, tc.reader, tc.bytesPerSecond)
		if err != nil {
			t.Error("got err:", err)
		}
		rlr, ok := reader.(rateLimitedReader)
		if !ok {
			t.Errorf("want type rateLimitedReader, got type %T", reader)
		}
		if rlr.ctx != ctx {
			t.Error("want same context")
		}
		if rlr.reader != tc.reader {
			t.Error("want same reader")
		}
		if rlr.bytesPerSecond != tc.bytesPerSecond {
			t.Errorf("want bytesPerSecond %v, got %v", tc.bytesPerSecond, rlr.bytesPerSecond)
		}
		if rlr.limiter.Limit() != tc.wantLimit {
			t.Errorf("want rate limit %v, got %v", tc.wantLimit, rlr.limiter.Limit())
		}
		if rlr.limiter.Burst() != tc.wantBurst {
			t.Errorf("want burst limit %v, got %v", tc.wantBurst, rlr.limiter.Burst())
		}
	}
}

func TestRateLimitedReaderReadNoBufferResize(t *testing.T) {
	ctx := context.Background()
	readBuf := make([]byte, 1000)
	reader := bytes.NewReader(readBuf)
	var bytesPerSecond rate.Limit = 1000 // One byte per millisecond.

	r, err := NewRateLimitedReader(ctx, reader, bytesPerSecond)
	if err != nil {
		t.Error("NewRateLimitedReader got err:", err)
	}

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
	readBuf := make([]byte, 1000)
	reader := bytes.NewReader(readBuf)
	var bytesPerSecond rate.Limit = 1000 // One byte per millisecond.

	r, err := NewRateLimitedReader(ctx, reader, bytesPerSecond)
	if err != nil {
		t.Error("NewRateLimitedReader got err:", err)
	}
	rlr, ok := r.(rateLimitedReader)
	if !ok {
		t.Errorf("want type rateLimitedReader, got type %T", reader)
	}
	rlr.chunksPerSecond = 100.0

	writeBuf := make([]byte, 1000)

	start := time.Now()
	// Since we've overridden the chunksPerSecond to be 100, we expect Read
	// to yield 10 bytes and take ~10ms.
	n, err := rlr.Read(writeBuf)
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
