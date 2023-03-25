package storage

import (
  "sync"
  "sync/atomic"
)

type Cache[T any] struct {
  lock    sync.Mutex
  items   []T
  counter atomic.Uint32
}

func NewCache[T any]() *Cache[T] {
  return &Cache[T]{
    counter: atomic.Uint32{},
  }
}

func (c *Cache[T]) Size() int {
  return int(c.counter.Load())
}

func (c *Cache[T]) Put(item T) {
  c.lock.Lock()
  defer c.lock.Unlock()

  c.items = append(c.items, item)
  c.counter.Add(uint32(1))
}

func (c *Cache[T]) Get() []T {
  c.lock.Lock()
  defer c.lock.Unlock()

  return c.items
}

func (c *Cache[T]) Flush() {
  c.lock.Lock()
  defer c.lock.Unlock()

  c.items = c.items[:0]
  c.counter.Store(0)
}
