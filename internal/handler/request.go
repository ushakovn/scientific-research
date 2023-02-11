package handler

import (
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "scientific-research/pkg/utils/requests"
  "scientific-research/pkg/utils/slice"

  log "github.com/sirupsen/logrus"
)

type Request interface {
  *GetRequest
}

func readRequest[R Request](r *http.Request, req R) error {
  if r.Method == http.MethodGet {
    query := requests.PlainRequestQuery(r.URL.Query())
    b, err := json.Marshal(query)
    if err != nil {
      return fmt.Errorf("cannot convert to json: %v", err)
    }
    if err = json.Unmarshal(b, req); err != nil {
      return fmt.Errorf("cannot parse json: %v", err)
    }
    return nil
  }
  if r.Method == http.MethodPost {
    b, err := io.ReadAll(r.Body)
    if err != nil {
      return fmt.Errorf("cannot read request body: %v", err)
    }
    defer func() {
      if err := r.Body.Close(); err != nil {
        log.Errorf("cannot close request body: %v", err)
      }
    }()
    if err = json.Unmarshal(b, req); err != nil {
      return fmt.Errorf("cannot parse json: %v", err)
    }
    return nil
  }
  return fmt.Errorf("unsupported request method: %s", r.Method)
}

func writeError(w http.ResponseWriter, err error, code int) {
  http.Error(w, err.Error(), code)
}

func writeResponse(w http.ResponseWriter, resp []byte, headers ...requests.ResponseHeaders) error {
  requests.ApplyResponseHeaders(w, slice.ExtractOptional(headers...))
  w.WriteHeader(http.StatusOK)
  _, err := w.Write(resp)
  return err
}