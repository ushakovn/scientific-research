package handler

import (
  "fmt"
  "scientific-research/internal/domain"
  "scientific-research/pkg/utils/convert"
  "scientific-research/pkg/utils/requests"
)

const (
  formatCsv  = "csv"
  formatJson = "json"
)

type GetRequest struct {
  Ticker string `json:"ticker"`
  Format string `json:"format"`
}

type GetResponse struct {
  Stocks []*domain.Stock
}

func (r *GetRequest) Validate() error {
  if r.Format == "" {
    r.Format = formatCsv
  } else
  if r.Format != formatCsv && r.Format != formatJson {
    return fmt.Errorf("invalid format: %v", r.Format)
  }
  return nil
}

func (r *GetResponse) Marshal(format string) ([]byte, error) {
  if format == formatCsv {
    return convert.ToCsvBytes(r.Stocks)
  }
  if format == formatJson {
    return convert.ToJsonBytes(r)
  }
  return nil, nil
}

func (r *GetRequest) FormResponseHeaders() requests.ResponseHeaders {
  if r.Format == formatCsv {
    return requests.ResponseHeaders{
      "Content-Type":        "text/csv",
      "Content-Disposition": "attachment;filename=out.csv",
    }
  }
  if r.Format == formatJson {
    return requests.ResponseHeaders{
      "Content-Type": "application/json",
    }
  }
  return nil
}
