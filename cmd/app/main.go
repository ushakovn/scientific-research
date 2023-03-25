package main

import (
  "context"
  "flag"
  "net/http"
  "os"
  "os/signal"
  "scientific-research/internal/fetcher/fetchers/polygon"
  "scientific-research/pkg/utils/httputils"
  "syscall"

  log "github.com/sirupsen/logrus"
)

func main() {
  ctx := context.Background()

  servePort := flag.String("port", "8080", "serving port")
  configPath := flag.String("path", "", "path to service config file")
  tickerId := flag.String("ticker", "", "specified ticker id for fetching")
  flag.Parse()

  cfg := polygon.NewConfig()
  if err := cfg.Parse(*configPath); err != nil {
    log.Fatalf("cannot parse fetcher config: %v", err)
  }

  fetcher, err := polygon.NewFetcher(ctx, cfg)
  if err != nil {
    log.Fatalf("cannot create new fetcher: %v", err)
  }
  if *tickerId != "" {
    fetcher.SetTickerId(*tickerId)
  }

  http.Handle("/health", httputils.HandleHealth())
  go httputils.ContinuouslyServe(*servePort)

  go fetcher.ContinuouslyFetch()
  defer fetcher.SaveFetcherState()

  serviceShutdown()
}

func serviceShutdown() {
  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}
