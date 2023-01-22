package utils

import (
  "time"

  log "github.com/sirupsen/logrus"
)

const (
  retryCount   = 5
  waitInterval = 1 * time.Minute
)

type RetryOption struct {
  RetryCount        int
  RetryWaitInterval time.Duration
}

func extractRetryOption(options []*RetryOption) *RetryOption {
  var option *RetryOption
  for _, opt := range options {
    if opt == nil {
      continue
    }
    option = opt
  }
  return option
}

func createDefaultOption() *RetryOption {
  return &RetryOption{
    RetryCount:        retryCount,
    RetryWaitInterval: waitInterval,
  }
}

func (opt *RetryOption) setOption(curr *RetryOption) {
  if curr == nil {
    return
  }
  if curr.RetryCount <= 0 {
    return
  }
  opt.RetryCount = curr.RetryCount
  opt.RetryWaitInterval = curr.RetryWaitInterval
}

func DoWithRetry(handler func() error, options ...*RetryOption) error {
  retryOpt := createDefaultOption()
  retryOpt.setOption(extractRetryOption(options))

  var err error
  for tryIdx := 1; tryIdx <= retryOpt.RetryCount; tryIdx++ {

    if err = handler(); err != nil {
      log.Warnf("attempt: %d, error: %v", tryIdx, err)
      time.Sleep(retryOpt.RetryWaitInterval)
      continue
    }
    break
  }

  return err
}
