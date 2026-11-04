package login

import "context"

// limiter bounds how many callers hold it at once.
type limiter chan struct{}

func newLimiter(n int) limiter {
	return make(limiter, n)
}

// acquire waits until fewer than the limit hold the limiter, or returns
// ctx's error when ctx ends first.
func (l limiter) acquire(ctx context.Context) error {
	select {
	case l <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l limiter) release() {
	<-l
}
