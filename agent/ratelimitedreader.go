package agent

import (
	"context"
	"io"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"

	"golang.org/x/time/rate"
)

// rateLimitedReader implements the io.Reader interface. It wraps another
// io.reader and enforces bandwidth control on the Read function.
type rateLimitedReader struct {
	ctx          context.Context
	reader       io.Reader
	limiter      *rate.Limiter
	statsTracker *stats.Tracker
}

func NewRateLimitedReader(ctx context.Context, r io.Reader, l *rate.Limiter, st *stats.Tracker) io.Reader {
	return rateLimitedReader{
		ctx:          ctx,
		reader:       r,
		limiter:      l,
		statsTracker: st,
	}
}

func (rlr rateLimitedReader) Read(buf []byte) (n int, err error) {
	// Shrink the read buf if necessary. This ensures the read doesn't just
	// block for one massive copy, and instead hands out data every second.
	l := rlr.limiter.Limit()
	if rate.Limit(len(buf)) > l {
		buf = buf[0:int(l)]
	}
	// Perform the read.
	if n, err = rlr.reader.Read(buf); err != nil {
		return 0, err
	}
	// Wait to enforce the rate limit.
	if err := rlr.limiter.WaitN(rlr.ctx, n); err != nil {
		return 0, err
	}
	if rlr.statsTracker != nil {
		rlr.statsTracker.RecordBytesSent(int64(n))
	}
	return n, nil
}
