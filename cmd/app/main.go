package main

import (
  "context"
  "os"
  "os/signal"
  "scientific-research/internal/fetcher/fetchers/polygon"
  "scientific-research/internal/handler"
  "syscall"

  log "github.com/sirupsen/logrus"
)

func main() {
  ctx := context.Background()

  f, err := polygon.NewFetcher(ctx)
  if err != nil {
    log.Fatal(err)
  }

  go f.ContinuouslyFetch()

  h := handler.NewHandler(f)
  h.SetHandles()

  port := GetPort()
  go h.ContinuouslyServe(port)

  GracefulShutdown()
}

func GracefulShutdown() {
  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}

func GetPort() string {
  port := os.Getenv("PORT")
  if port == "" {
    return "8080"
  }
  return port
}
