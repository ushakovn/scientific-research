package handler

import (
  "fmt"
  "net/http"
  "scientific-research/internal/fetcher"

  log "github.com/sirupsen/logrus"
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
  req := &GetRequest{}

  if err := readRequest(r, req); err != nil {
    writeError(w, err, http.StatusBadRequest)
    return
  }
  if err := req.Validate(); err != nil {
    writeError(w, err, http.StatusBadRequest)
    return
  }

  stocks, err := h.fetcher.QueryFetchedStocks(req.Ticker)
  if err != nil {
    writeError(w, err, http.StatusInternalServerError)
    return
  }
  if len(stocks) == 0 {
    writeError(w, fmt.Errorf("stocks not found. try later"), http.StatusNotFound)
    return
  }

  response := GetResponse{
    Stocks: stocks,
  }
  resp, err := response.Marshal(req.Format)
  if err != nil {
    writeError(w, err, http.StatusInternalServerError)
    return
  }

  if err = writeResponse(w, resp, req.FormResponseHeaders()); err != nil {
    writeError(w, err, http.StatusInternalServerError)
    return
  }
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
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
