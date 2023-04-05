package dateutils

import (
	"context"
	"time"
)

// Waiter will return a channel that will close after the specified duration.
// If the context is cancelled, the channel will never close.
func Waiter(ctx context.Context, duration time.Duration, closeOnCtxDone bool) <-chan struct{} {
	return WaiterWithCallback(ctx, duration, closeOnCtxDone, nil)
}

// WaiterWithCallback will return a channel that will close after the specified duration.
// If the context is cancelled, the channel will never close.
func WaiterWithCallback(ctx context.Context, duration time.Duration, closeOnCtxDone bool, callback func(ctx context.Context)) <-chan struct{} {
	waitChan := make(chan struct{})
	timer := time.NewTimer(duration)
	go func() {
		select {
		case <-timer.C:
			close(waitChan)
			if callback != nil {
				callback(ctx)
			}
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			if closeOnCtxDone {
				close(waitChan)
			}
		}
	}()
	return waitChan
}
