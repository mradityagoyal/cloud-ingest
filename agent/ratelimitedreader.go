package agent

import (
	"context"
	"fmt"
	"io"
	"math"

	"golang.org/x/time/rate"
)

// rateLimitedReader implements the io.Reader interface. It wraps another
// io.reader and enforces bandwidth control on the Read function.
type rateLimitedReader struct {
	ctx            context.Context
	reader         io.Reader
	bytesPerSecond rate.Limit
	limiter        *rate.Limiter

	// Only accessible in testing. The buf written to by Read will be sized
	// to only yield at most bytesPerSecond/chunksPerSecond bytes per call.
	chunksPerSecond rate.Limit
}

func NewRateLimitedReader(ctx context.Context, r io.Reader, bytesPerSecond rate.Limit) (io.Reader, error) {
	l := rate.NewLimiter(bytesPerSecond, math.MaxInt32)
	if bytesPerSecond <= 0 {
		l = rate.NewLimiter(rate.Inf, math.MaxInt32)
	}
	if err := l.WaitN(ctx, math.MaxInt32); err != nil {
		return nil, fmt.Errorf("error draining new rate limiter, err: %v", err)
	}
	return rateLimitedReader{
		ctx:             ctx,
		reader:          r,
		limiter:         l,
		bytesPerSecond:  bytesPerSecond,
		chunksPerSecond: 1.0,
	}, nil
}

func (rlr rateLimitedReader) Read(buf []byte) (n int, err error) {
	// Shrink the read buf if necessary. This ensures the read doesn't just
	// block for one massive copy, and instead hands out data every
	// 1/chunksPerSecond seconds.
	if rlr.bytesPerSecond > 0 && rlr.chunksPerSecond > 0 && len(buf) > int(rlr.bytesPerSecond/rlr.chunksPerSecond) {
		buf = buf[0:int(rlr.bytesPerSecond/rlr.chunksPerSecond)]
	}
	// Perform the read.
	if n, err = rlr.reader.Read(buf); err != nil {
		return 0, err
	}
	// Wait to enforce the rate limit.
	if err := rlr.limiter.WaitN(rlr.ctx, n); err != nil {
		return 0, err
	}
	return n, nil
}
