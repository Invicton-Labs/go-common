package gensync

import (
	"fmt"
	"sync"

	"github.com/Invicton-Labs/go-common/collections"
	"github.com/Invicton-Labs/go-stackerr"
	"golang.org/x/sync/errgroup"
)

type MultiLock[T comparable] interface {
	LockAll(excluded ...T)
	UnlockAll(excluded ...T)
	Lock(key T)
	Unlock(key T)
}

type multiLock[T comparable] struct {
	allLock  sync.Mutex
	locks    Map[T, *sync.Mutex]
	lockKeys []T
}

func NewMultiLock[T comparable](keys []T) MultiLock[T] {
	ml := &multiLock[T]{
		lockKeys: collections.CopySlice(keys),
	}
	for _, k := range keys {
		if _, loaded := ml.locks.LoadOrStore(k, &sync.Mutex{}); loaded {
			panic(fmt.Sprintf("Duplicate key provided: %v", k))
		}

	}
	return ml
}

func (ml *multiLock[T]) Lock(key T) {
	lock, ok := ml.locks.Load(key)
	if !ok {
		panic(fmt.Sprintf("Lock key not found: %v", key))
	}
	lock.Lock()
}

func (ml *multiLock[T]) Unlock(key T) {
	lock, ok := ml.locks.Load(key)
	if !ok {
		panic(fmt.Sprintf("Lock key not found: %v", key))
	}
	lock.Unlock()
}

func (ml *multiLock[T]) LockAll(excluded ...T) {
	// We use an additional lock when trying to grab all locks,
	// which prevents competing all locks, which would result
	// in a deadlock.
	ml.allLock.Lock()
	defer ml.allLock.Unlock()

	excludedKeys := map[T]struct{}{}
	for _, ek := range excluded {
		excludedKeys[ek] = struct{}{}
	}
	errgrp := errgroup.Group{}
	for _, k := range ml.lockKeys {
		if _, ok := excludedKeys[k]; !ok {
			lockLey := k
			errgrp.Go(func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = stackerr.FromRecover(r)
					}
				}()
				l, _ := ml.locks.Load(lockLey)
				l.Lock()
				return nil
			})
		}
	}
	if err := errgrp.Wait(); err != nil {
		panic(err)
	}
}

func (ml *multiLock[T]) UnlockAll(excluded ...T) {
	excludedKeys := map[T]struct{}{}
	for _, ek := range excluded {
		excludedKeys[ek] = struct{}{}
	}
	for _, k := range ml.lockKeys {
		if _, ok := excludedKeys[k]; !ok {
			l, _ := ml.locks.Load(k)
			l.Unlock()
		}
	}
}
