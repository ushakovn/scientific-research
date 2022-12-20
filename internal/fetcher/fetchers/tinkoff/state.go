package tinkoff // Package tinkoff: unused

import (
  "time"
  "sync/atomic"
  log "github.com/sirupsen/logrus"
  "scientific-research/internal/domain"
)

const (
  retryCount = 5

  recentlyFetchedSleepInterval  = 1 * time.Hour
  encounteredErrorSleepInterval = 10 * time.Minute
)

type state struct {
  lastUpd   time.Time
  lastCount atomic.Uint32
}

func (f *Fetcher) countInc(count int) {
  fetched := uint32(count)
  f.totalCount.Add(fetched)
  f.state.lastCount.Add(fetched)
  f.state.lastUpd = time.Now()
}

func (f *Fetcher) getTotalCount() int {
  return int(f.totalCount.Load())
}

func (f *Fetcher) countReset() {
  f.totalCount.Swap(0)
  f.state.lastCount.Swap(0)
  f.state.lastUpd = time.Now()
}

func (f *Fetcher) HasRelevantState() bool {
  thresholdTime := f.state.lastUpd.Add(24 * time.Hour)
  return f.state.lastCount.Load() != 0 && thresholdTime.After(time.Now())
}

func (f *Fetcher) ContinuouslyFetch() {
  tryLeft := retryCount

  for tryLeft >= 0 {
    if f.HasRelevantState() {
      log.Printf("recently fetched. wait %v before the next launch",
        recentlyFetchedSleepInterval)
      time.Sleep(recentlyFetchedSleepInterval)
      continue
    }

    err := f.fetchPaginatedStocks()
    if err != nil {
      log.Errorf("fetching error: %v. wait %v before the next launch",
        err, encounteredErrorSleepInterval)
      time.Sleep(encounteredErrorSleepInterval)

      tryLeft--
      continue
    }
    totalFetched := f.getTotalCount()

    log.Printf("succefully fetched %d stocks", totalFetched)
  }
  log.Fatalf("fetching failed and stopped")
}

func (f *Fetcher) SetTicker(ticker string) {
  panic("implement me")
}

func (f *Fetcher) QueryFetchedStocks(ticker string) ([]*domain.Stock, error) {
  panic("implement me")
}
