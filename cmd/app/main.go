package main

import (
  "context"
  "os"
  "os/signal"
  "syscall"
  "scientific-research/internal/fetcher/fetchers/polygon"
  log "github.com/sirupsen/logrus"
  "scientific-research/internal/handler"
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

  port := getPort()
  go h.ContinuouslyServe(port)

  hold()
}

func hold() {
  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}

func getPort() string {
  port := os.Getenv("PORT")
  if port == "" {
    return "8080"
  }
  return port
}
