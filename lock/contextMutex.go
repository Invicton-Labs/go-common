package lock

import (
	"context"

	"github.com/Invicton-Labs/go-stackerr"
)

// CtxMutex is a mutex where Lock operations use a context
// that can be cancelled/deadlined to terminate the lock
// attempt.
type CtxMutex interface {

	// Lock will wait until either the mutex can be locked, or
	// the context is done (cancelled/deadlined). If the lock
	// succeeds, it will return nil. If the context is done,
	// it will return a stack-wrapped version of the context's error.
	Lock(ctx context.Context) (err stackerr.Error)

	// TryLock will attempt to lock the mutex, but will not
	// wait if it cannot immediately do so.
	TryLock() (locked bool)

	// Unlock will unlock the mutex, and will panic if the mutex
	// is not currently locked.
	Unlock()

	// TryUnlock will attempt to unlock the mutex, but will not
	// panic if the mutex is not currently locked.
	TryUnlock() (unlocked bool)
}

type ctxMutex struct {
	ch chan struct{}
}

// NewCtxMutex creates a new CtxMutex
func NewCtxMutex() CtxMutex {
	return &ctxMutex{
		ch: make(chan struct{}, 1),
	}
}

func (mu *ctxMutex) Lock(ctx context.Context) (err stackerr.Error) {
	select {
	case <-ctx.Done():
		return stackerr.Wrap(ctx.Err())
	case mu.ch <- struct{}{}:
		return nil
	}
}

func (mu *ctxMutex) TryLock() (locked bool) {
	select {
	case mu.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (mu *ctxMutex) Unlock() {
	select {
	case <-mu.ch:
		return
	default:
		panic("unlock of unlocked mutex")
	}
}

func (mu *ctxMutex) TryUnlock() (unlocked bool) {
	select {
	case <-mu.ch:
		return true
	default:
		return false
	}
}

// Locked will return whether the mutex is currently locked.
func (mu *ctxMutex) Locked() (locked bool) {
	return len(mu.ch) > 0
}
