package polygon

import (
  "scientific-research/internal/domain"
  "sync/atomic"
  "time"

  log "github.com/sirupsen/logrus"
)

const (
  retryCount = 5

  recentlyFetchedSleepInterval  = 1 * time.Hour
  encounteredErrorSleepInterval = 10 * time.Minute

  modeTotal   = 0
  modeCurrent = 1
)

type state struct {
  lastUpd   time.Time
  lastCount atomic.Uint32
  lastMode  int
}

func (f *Fetcher) setMode(mode int) {
  if mode != modeTotal && mode != modeCurrent {
    log.Warnf("invalid mode code: %d. mode do not changed", mode)
    return
  }
  f.state.lastMode = mode
  log.Printf("set fetcher mode: %d. possible: %d - total, %d - current",
    mode, modeTotal, modeCurrent)
}

func (f *Fetcher) SetTicker(ticker string) {
  if ticker == "" {
    log.Warnf("empty ticker name. ticker do not set")
  }
  f.ticker = &domain.Ticker{
    Ticker: ticker,
  }
}

func (f *Fetcher) countInc(count int) {
  fetched := uint32(count)
  f.state.lastCount.Add(fetched)
  f.state.lastUpd = time.Now()
}

func (f *Fetcher) countReset() {
  f.state.lastCount.Swap(0)
  f.state.lastUpd = time.Now()
}

func (f *Fetcher) hasRelevantState() bool {
  thresholdTime := f.state.lastUpd.Add(24 * time.Hour)
  return f.state.lastCount.Load() != 0 && thresholdTime.After(time.Now())
}

func (f *Fetcher) hasSpeciallyTicker() bool {
  return f.ticker != nil
}

func (f *Fetcher) ContinuouslyFetch() {
  var err error

  f.setMode(modeTotal)
  tryLeft := retryCount

  for tryLeft >= 0 {
    if f.hasRelevantState() {

      log.Printf("recently fetched. wait %v before the next launch",
        recentlyFetchedSleepInterval)

      time.Sleep(recentlyFetchedSleepInterval)
      continue
    }

    if f.hasSpeciallyTicker() {
      err = f.fetchStocks(f.ticker)
    } else {
      err = f.fetchTickers(f.fetchStocks)
    }
    if err != nil {
      log.Errorf("fetching error: %v. wait %v before the next launch",
        err, encounteredErrorSleepInterval)

      time.Sleep(encounteredErrorSleepInterval)

      tryLeft--
      continue
    }

    log.Println("successfully fetching finished")
    if f.hasSpeciallyTicker() {
      return
    }

    f.once.Do(func() {
      f.setMode(modeCurrent)
    })
  }
  log.Fatalf("fetching failed and stopped")
}

func (f *Fetcher) QueryFetchedStocks(ticker string) ([]*domain.Stock, error) {
  return f.stocksCache.QueryStocks(ticker)
}
