package copy

import (
	"context"
	"flag"
	"io"

	"golang.org/x/sync/semaphore"
)

var (
	globalReadSem *semaphore.Weighted

	concurrentReadMax = flag.Int("concurrent-read-max", -1, "The maximum allowed number of concurrent file Read calls. A negative value means there is no maximum. Zero will block all reads.")
)

// SemAcquiringReader is an io.Reader that wraps another io.Reader and limits the
// number of concurrent Read calls globally.
type SemAcquiringReader struct {
	reader io.Reader
	ctx    context.Context
}

// NewSemAcquiringReader returns a SemAcquiringReader. If Read concurrency is unlimited
// then this will return the passed in io.Reader.
func NewSemAcquiringReader(r io.Reader, ctx context.Context) io.Reader {
	if *concurrentReadMax < 0 {
		return r
	}
	if globalReadSem == nil {
		globalReadSem = semaphore.NewWeighted(int64(*concurrentReadMax))
	}
	return &SemAcquiringReader{reader: r, ctx: ctx}
}

// Read implements the io.Reader interface.
func (sar *SemAcquiringReader) Read(buf []byte) (n int, err error) {
	globalReadSem.Acquire(sar.ctx, 1)
	defer globalReadSem.Release(1)
	if n, err = sar.reader.Read(buf); err != nil {
		return 0, err
	}
	return n, nil
}
