package retries

import (
  "scientific-research/pkg/utils/slice"
  "time"

  log "github.com/sirupsen/logrus"
)

const (
  retryCount   = 5
  waitInterval = 1 * time.Minute
)

type HandlerFuncE func() error

type HandlerFunc func()

type Option struct {
  RetryCount   int
  WaitInterval time.Duration
}

func NewDefaultOption() *Option {
  return &Option{
    RetryCount:   retryCount,
    WaitInterval: waitInterval,
  }
}

func (o *Option) TrySet(option *Option) {
  if option == nil {
    return
  }
  if option.RetryCount <= 0 {
    return
  }
  o.RetryCount = option.RetryCount
  o.WaitInterval = option.WaitInterval
}

func DoWithRetry(h HandlerFuncE, options ...*Option) error {
  option := slice.ExtractOptional(options...)

  retryOption := NewDefaultOption()
  retryOption.TrySet(option)

  var err error

  for tryIdx := 1; tryIdx <= retryOption.RetryCount; tryIdx++ {

    if err = h(); err != nil {
      log.Warnf("try: %d, error: %v", tryIdx, err)

      time.Sleep(retryOption.WaitInterval)
      continue
    }
    break
  }

  return err
}

func DoWithRepeat(repeatCount int, h HandlerFunc) {
  for runIdx := 1; runIdx <= repeatCount; runIdx++ {
    h()
  }
}
