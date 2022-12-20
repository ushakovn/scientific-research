package storage

import (
  "sync/atomic"
  "fmt"
  "scientific-research/internal/domain"
  "scientific-research/pkg/utils"
  log "github.com/sirupsen/logrus"
)

const batchSize = 25

type CacheStorage struct {
  *Storage
  cache       []any
  counter     atomic.Uint32
  batchSize   int
  description string
}

func NewCache(storage *Storage, desc string) (*CacheStorage, error) {
  return &CacheStorage{
    Storage:     storage,
    counter:     atomic.Uint32{},
    batchSize:   batchSize,
    description: desc,
  }, nil
}

func (c *CacheStorage) SetBatchSize(size int) error {
  if size <= 0 {
    return fmt.Errorf("non positive batch size: %d", size)
  }
  c.batchSize = size
  return nil
}

func (c *CacheStorage) getCount() int {
  return int(c.counter.Load())
}

func (c *CacheStorage) countInc(count int) {
  c.counter.Add(uint32(count))
}

func (c *CacheStorage) Put(d any) error {
  if d == nil {
    return fmt.Errorf("invalid empty data")
  }
  c.cache = append(c.cache, d)

  if len(c.cache) >= c.batchSize {
    err := c.putBatch(c.cache)
    if err != nil {
      return err
    }
    c.countInc(c.batchSize)
    c.cache = c.cache[:0]

    log.Printf("%s: current stored %d, total stored: %d",
      c.description, c.batchSize, c.getCount())
  }

  return nil
}

func (c *CacheStorage) putBatch(d []any) error {
  var err error

  for _, dI := range d {
    switch t := dI.(type) {

    case *domain.Stock:
      var stocks []*domain.Stock
      if err := utils.MapFields(d, &stocks); err != nil {
        return err
      }
      err = c.PutBatchStocks(stocks)

    case *domain.Ticker:
      var tickers []*domain.Ticker
      if err := utils.MapFields(d, &tickers); err != nil {
        return err
      }
      err = c.PutBatchTickers(tickers)

    default:
      err = fmt.Errorf("%s: unsupported domain type: %T", c.description, t)
    }
    break
  }

  return err
}
