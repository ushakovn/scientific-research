package handler

import (
  "net/http"
  log "github.com/sirupsen/logrus"
  "scientific-research/internal/fetcher"
  "scientific-research/pkg/utils"
)

type Handler struct {
  fetcher fetcher.Fetcher
}

func NewHandler(f fetcher.Fetcher) *Handler {
  return &Handler{
    fetcher: f,
  }
}

func (h *Handler) SetHandles() {
  http.Handle("/get", http.HandlerFunc(h.handleGet))
  http.Handle("/health", http.HandlerFunc(h.handleHealth))
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {

  query := r.URL.Query()
  ticker := query.Get("ticker")

  res, err := h.fetcher.QueryFetchedStocks(ticker)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  var resIf []any
  for _, v := range res {
    resIf = append(resIf, v)
  }
  csv, err := utils.CreateCSV(resIf)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  if len(csv) == 0 {
    http.Error(w, "stocks not found. try later", http.StatusNotFound)
    return
  }

  w.Header().Add("Content-Type", "text/csv")
  w.Header().Add("Content-Disposition", "attachment;filename=out.csv")
  w.WriteHeader(http.StatusOK)

  if _, err := w.Write(csv); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusOK)
  _, err := w.Write([]byte("/ok"))
  if err != nil {
    log.Printf("cannot write to response writer: %v", err)
  }
}

func (h *Handler) ContinuouslyServe(port string) {
  var handler http.Handler
  err := http.ListenAndServe(":"+port, handler)
  if err != nil {
    log.Fatalf("listen and serve error: %v", err)
  }
}
