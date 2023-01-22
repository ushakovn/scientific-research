package polygon

import (
  "fmt"
  "os"
  "scientific-research/internal/domain"
  "strconv"
  "sync/atomic"
  "time"

  log "github.com/sirupsen/logrus"
)

type state struct {
  lastUpd          time.Time
  lastCount        atomic.Uint32
  lastModeCode     int
  modeTotalHours   int
  modeCurrentHours int
}

func initState() (*state, error) {
  modeTotalHours, err := strconv.Atoi(os.Getenv(modeTotalHoursEnv))
  if err != nil {
    return nil, fmt.Errorf("cannot convert hours for total mode: %v", err)
  }

  modeCurrentHours, err := strconv.Atoi(os.Getenv(modeCurrentHoursEnv))
  if err != nil {
    return nil, fmt.Errorf("cannot convert hours for current mode: %v", err)
  }

  return &state{
    lastModeCode:     fetcherModeTotal,
    modeTotalHours:   modeTotalHours,
    modeCurrentHours: modeCurrentHours,
  }, nil
}

func (f *Fetcher) setModeCode(mode int) {
  if mode != fetcherModeTotal && mode != fetcherModeCurrent {
    log.Warnf("invalid fetcher mode code: %d. mode do not changed", mode)
    return
  }
  f.state.lastModeCode = mode
  log.Printf("set fetcher mode: %d. possible: %d - total, %d - current",
    mode, fetcherModeTotal, fetcherModeCurrent)
}

func (f *Fetcher) SetTicker(tickerName string) {
  if tickerName == "" {
    log.Warnf("empty ticker name. ticker do not set")
    return
  }
  f.ticker = &domain.Ticker{
    Ticker: tickerName,
  }
}

func (f *Fetcher) countInc(count int) {
  fetched := uint32(count)
  f.state.lastCount.Add(fetched)
  f.state.lastUpd = time.Now()
}

func (f *Fetcher) countReset() {
  f.state.lastCount.Store(0)
  f.state.lastUpd = time.Now()
}

func (f *Fetcher) hasRelevantState() bool {
  thresholdTime := f.state.lastUpd.Add(relevantThresholdInterval)
  return f.state.lastCount.Load() != 0 && thresholdTime.After(time.Now())
}

func (f *Fetcher) hasSpeciallyTicker() bool {
  return f.ticker != nil
}

func (f *Fetcher) ContinuouslyFetch() {
  f.setModeCode(fetcherModeTotal)

  tryLeft := fetcherRetryCount

  for tryLeft >= 0 {
    if f.hasRelevantState() {

      log.Printf("recently fetched. wait %v before the next fetch",
        recentlyFetchedSleepInterval)

      time.Sleep(recentlyFetchedSleepInterval)
      continue
    }

    var err error

    if f.hasSpeciallyTicker() {
      err = f.fetchStocks(f.ticker)

    } else {
      err = f.fetchTickers(f.fetchStocks)
    }

    if err != nil {
      log.Errorf("fetching error: %v. wait %v before the next fetch",
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
      f.setModeCode(fetcherModeCurrent)
    })
  }

  log.Fatalf("fetching failed and stopped")
}

func (f *Fetcher) QueryFetchedStocks(ticker string) ([]*domain.Stock, error) {
  return f.storage.QueryStocks(ticker)
}
