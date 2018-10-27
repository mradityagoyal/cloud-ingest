package agent

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// rateLimitedReader implements the io.Reader interface. It wraps another
// io.reader and enforces bandwidth control on the Read function.
type rateLimitedReader struct {
	ctx     context.Context
	reader  io.Reader
	limiter *rate.Limiter
}

func NewRateLimitedReader(ctx context.Context, r io.Reader, l *rate.Limiter) io.Reader {
	return rateLimitedReader{
		ctx:     ctx,
		reader:  r,
		limiter: l,
	}
}

func (rlr rateLimitedReader) Read(buf []byte) (n int, err error) {
	// Shrink the read buf if necessary. This ensures the read doesn't just
	// block for one massive copy, and instead hands out data every second.
	if len(buf) > int(rlr.limiter.Limit()) {
		buf = buf[0:int(rlr.limiter.Limit())]
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
